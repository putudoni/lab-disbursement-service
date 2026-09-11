package xendit

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"lab-disbursement-service/pkg/pg_provider"
)

type fakePoster struct {
	mu          atomic.Int32
	statuses    []int
	bodies      []string
	errs        []error
	lastPath    string
	lastHeaders map[string]string
	lastBody    []byte
	delay       time.Duration
}

func (f *fakePoster) PostWithHeaders(ctx context.Context, path string, queryParams map[string]any,
	body []byte, headers map[string]string) (*http.Response, error) {
	i := int(f.mu.Add(1)) - 1

	f.lastPath = path
	f.lastHeaders = headers
	f.lastBody = body

	if f.delay > 0 {
		time.Sleep(f.delay)
	}

	status := http.StatusOK
	if i < len(f.statuses) {
		status = f.statuses[i]
	}

	var err error
	if i < len(f.errs) {
		err = f.errs[i]
	}

	if err != nil {
		return nil, err
	}

	bodyContent := `{"id":"vid-1","status":"verified","name_match_result":"MATCH","account_number":"11111111","bank_code":"BCA","name":"Budi"}`
	if i < len(f.bodies) {
		bodyContent = f.bodies[i]
	}

	rec := httptest.NewRecorder()
	rec.WriteHeader(status)
	_, _ = rec.WriteString(bodyContent)

	return rec.Result(), nil
}

func newClient(f *fakePoster) *Client {
	return NewClient(Config{
		APIKey:         "key",
		APISecret:      "secret",
		RetryMax:       2,
		RetryBaseDelay: time.Millisecond,
		RetryMaxDelay:  time.Millisecond,
	}, f)
}

func validationReq() pg_provider.BankAccountValidationRequest {
	return pg_provider.BankAccountValidationRequest{
		BankAccountNumber: "11111111",
		BankCode:          "BCA",
		AccountHolderName: "Budi",
	}
}

func TestValidate_success(t *testing.T) {
	p := &fakePoster{}
	c := newClient(p)

	resp, err := c.Validate(context.Background(), validationReq())

	require.NoError(t, err)
	require.Equal(t, "vid-1", resp.ID)
	require.Equal(t, "verified", resp.Status)
	require.Equal(t, int32(1), p.mu.Load())
	require.Equal(t, requestPath, p.lastPath)
	require.Equal(t, authorizationBasic+base64.StdEncoding.EncodeToString([]byte("key:secret")), p.lastHeaders["Authorization"])
}

func TestValidate_retriesOnServerError(t *testing.T) {
	p := &fakePoster{statuses: []int{500, 502}}
	c := newClient(p)

	resp, err := c.Validate(context.Background(), validationReq())

	require.NoError(t, err)
	require.Equal(t, "vid-1", resp.ID)
	require.Equal(t, int32(3), p.mu.Load())
}

func TestValidate_maxRetriesExhaustedOnServerError(t *testing.T) {
	p := &fakePoster{statuses: []int{500, 502, 503}}
	c := newClient(p)

	_, err := c.Validate(context.Background(), validationReq())

	require.Error(t, err)
	require.True(t, pg_provider.IsTransient(err))
	require.Equal(t, int32(3), p.mu.Load())
}

func TestValidate_retriesOnNetworkError(t *testing.T) {
	p := &fakePoster{errs: []error{errors.New("connection refused"), errors.New("timeout")}}
	c := newClient(p)

	_, err := c.Validate(context.Background(), validationReq())

	require.NoError(t, err)
	require.Equal(t, int32(3), p.mu.Load())
}

func TestValidate_noRetryOnClientError(t *testing.T) {
	p := &fakePoster{
		statuses: []int{404},
		bodies:   []string{`{"error_code":"BANK_ACCOUNT_NOT_FOUND","message":"account not found"}`},
	}
	c := newClient(p)

	_, err := c.Validate(context.Background(), validationReq())

	require.Error(t, err)
	require.True(t, pg_provider.IsPermanent(err))
	require.Equal(t, int32(1), p.mu.Load())

	var pe *pg_provider.ProviderError
	require.ErrorAs(t, err, &pe)
	require.Equal(t, 404, pe.StatusCode)
	require.Equal(t, "BANK_ACCOUNT_NOT_FOUND", pe.ErrorCode)
	require.Equal(t, "account not found", pe.Message)
}

func TestValidate_unverifiableBodyFallsBackToUnknownError(t *testing.T) {
	p := &fakePoster{
		statuses: []int{400},
		bodies:   []string{"<html>bad request</html>"},
	}
	c := newClient(p)

	_, err := c.Validate(context.Background(), validationReq())

	require.Error(t, err)
	require.True(t, pg_provider.IsPermanent(err))

	var pe *pg_provider.ProviderError
	require.ErrorAs(t, err, &pe)
	require.Equal(t, 400, pe.StatusCode)
	require.Equal(t, "UNKNOWN_ERROR", pe.ErrorCode)
}
