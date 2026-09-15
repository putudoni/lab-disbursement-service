package xendit

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"lab-disbursement-service/pkg/pg_provider"
)

func disbursementReq() pg_provider.DisbursementRequest {
	return pg_provider.DisbursementRequest{
		ExternalID:        "dsb-1",
		BankCode:          "BCA",
		AccountNumber:     "11111111",
		AccountHolderName: "Budi",
		Description:       "Pencairan pinjaman",
		Amount:            500000,
	}
}

const disbursementCompletedBody = `{"id":"dsb-trx-1","external_id":"dsb-1","status":"COMPLETED","amount":500000}`

func TestDisburse_success(t *testing.T) {
	p := &fakePoster{
		bodies: []string{disbursementCompletedBody},
	}
	c := newClient(p)

	resp, err := c.Disburse(context.Background(), disbursementReq())

	require.NoError(t, err)
	require.Equal(t, "dsb-trx-1", resp.ID)
	require.Equal(t, "dsb-1", resp.ExternalID)
	require.Equal(t, pg_provider.DisbursementStatusCompleted, resp.Status)
	require.Equal(t, int64(500000), resp.Amount)
	require.Equal(t, int32(1), p.mu.Load())
	require.Equal(t, disbursementPath, p.lastPath)
	require.Equal(t, authorizationBasic+base64.StdEncoding.EncodeToString([]byte("xnd_development_secret:")), p.lastHeaders["Authorization"])
	require.Equal(t, "dsb-1", p.lastHeaders["X-Idempotency-Key"])
	require.Contains(t, string(p.lastBody), `"description":"Pencairan pinjaman"`)
	require.Contains(t, string(p.lastBody), `"external_id":"dsb-1"`)
}

func TestDisburse_pendingStatus(t *testing.T) {
	p := &fakePoster{
		bodies: []string{`{"id":"dsb-trx-2","external_id":"dsb-1","status":"PENDING","amount":100000}`},
	}
	c := newClient(p)

	resp, err := c.Disburse(context.Background(), disbursementReq())

	require.NoError(t, err)
	require.Equal(t, pg_provider.DisbursementStatusPending, resp.Status)
	require.Equal(t, int32(1), p.mu.Load())
}

func TestDisburse_retriesOnServerError(t *testing.T) {
	p := &fakePoster{
		statuses: []int{500, 502},
		bodies:   []string{`{"error_code":"PROVIDER_UNAVAILABLE","message":"down"}`, `{"error_code":"PROVIDER_UNAVAILABLE","message":"down"}`, disbursementCompletedBody},
	}
	c := newClient(p)

	resp, err := c.Disburse(context.Background(), disbursementReq())

	require.NoError(t, err)
	require.Equal(t, pg_provider.DisbursementStatusCompleted, resp.Status)
	require.Equal(t, int32(3), p.mu.Load())
}

func TestDisburse_maxRetriesExhaustedOnServerError(t *testing.T) {
	p := &fakePoster{statuses: []int{500, 502, 503}}
	c := newClient(p)

	_, err := c.Disburse(context.Background(), disbursementReq())

	require.Error(t, err)
	require.True(t, pg_provider.IsTransient(err))
	require.Equal(t, int32(3), p.mu.Load())
}

func TestDisburse_noRetryOnClientError(t *testing.T) {
	p := &fakePoster{
		statuses: []int{400},
		bodies:   []string{`{"error_code":"INSUFFICIENT_BALANCE","message":"balance not enough"}`},
	}
	c := newClient(p)

	_, err := c.Disburse(context.Background(), disbursementReq())

	require.Error(t, err)
	require.True(t, pg_provider.IsPermanent(err))
	require.Equal(t, int32(1), p.mu.Load())

	var pe *pg_provider.ProviderError
	require.ErrorAs(t, err, &pe)
	require.Equal(t, 400, pe.StatusCode)
	require.Equal(t, "INSUFFICIENT_BALANCE", pe.ErrorCode)
	require.Equal(t, "balance not enough", pe.Message)
}

func TestDisburse_unverifiableBodyFallsBackToUnknownError(t *testing.T) {
	p := &fakePoster{
		statuses: []int{422},
		bodies:   []string{"<html>unprocessable</html>"},
	}
	c := newClient(p)

	_, err := c.Disburse(context.Background(), disbursementReq())

	require.Error(t, err)
	require.True(t, pg_provider.IsPermanent(err))

	var pe *pg_provider.ProviderError
	require.ErrorAs(t, err, &pe)
	require.Equal(t, 422, pe.StatusCode)
	require.Equal(t, "UNKNOWN_ERROR", pe.ErrorCode)
}

func TestDisburse_retriesOnNetworkError(t *testing.T) {
	p := &fakePoster{
		errs:   []error{errors.New("connection refused"), errors.New("timeout")},
		bodies: []string{"", "", disbursementCompletedBody},
	}
	c := newClient(p)

	resp, err := c.Disburse(context.Background(), disbursementReq())

	require.NoError(t, err)
	require.Equal(t, pg_provider.DisbursementStatusCompleted, resp.Status)
	require.Equal(t, int32(3), p.mu.Load())
}

func TestDisburse_requestTimeout(t *testing.T) {
	p := &fakePoster{delay: 200 * time.Millisecond}
	c := newClient(p)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := c.Disburse(ctx, disbursementReq())

	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
