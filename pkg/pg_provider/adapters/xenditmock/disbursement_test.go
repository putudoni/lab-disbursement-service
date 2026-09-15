package xenditmock

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"lab-disbursement-service/pkg/pg_provider"
)

func disbursementReq(accountNumber string, externalID string) pg_provider.DisbursementRequest {
	return pg_provider.DisbursementRequest{
		ExternalID:        externalID,
		BankCode:          "BCA",
		AccountNumber:     accountNumber,
		AccountHolderName: "Budi",
		Description:       "Pencairan pinjaman",
		Amount:            500000,
	}
}

func TestDisburse_completed(t *testing.T) {
	c := NewClient(Config{})

	resp, err := c.Disburse(context.Background(), disbursementReq("11111111", "dsb-1"))

	require.NoError(t, err)
	require.Equal(t, pg_provider.DisbursementStatusCompleted, resp.Status)
	require.Equal(t, "dsb-1", resp.ExternalID)
	require.Equal(t, "dsb-trx-dsb-1", resp.ID)
	require.Equal(t, int64(500000), resp.Amount)
}

func TestDisburse_pendingThenCompleted(t *testing.T) {
	c := NewClient(Config{})
	req := disbursementReq("22222222", "dsb-2")

	first, err := c.Disburse(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, pg_provider.DisbursementStatusPending, first.Status)

	second, err := c.Disburse(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, pg_provider.DisbursementStatusCompleted, second.Status)
}

func TestDisburse_failed(t *testing.T) {
	c := NewClient(Config{})

	resp, err := c.Disburse(context.Background(), disbursementReq("99999999", "dsb-3"))

	require.NoError(t, err)
	require.Equal(t, pg_provider.DisbursementStatusFailed, resp.Status)
	require.Equal(t, "INVALID_DESTINATION", resp.FailureCode)
}

func TestDisburse_missingDescriptionIsPermanentError(t *testing.T) {
	c := NewClient(Config{})

	req := disbursementReq("11111111", "dsb-1")
	req.Description = ""

	_, err := c.Disburse(context.Background(), req)

	require.Error(t, err)
	require.True(t, pg_provider.IsPermanent(err))

	var pe *pg_provider.ProviderError
	require.ErrorAs(t, err, &pe)
	require.Equal(t, "API_VALIDATION_ERROR", pe.ErrorCode)
}

func TestDisburse_missingFieldsIsPermanentError(t *testing.T) {
	c := NewClient(Config{})

	_, err := c.Disburse(context.Background(), pg_provider.DisbursementRequest{})

	require.Error(t, err)
	require.True(t, pg_provider.IsPermanent(err))

	var pe *pg_provider.ProviderError
	require.ErrorAs(t, err, &pe)
	require.Equal(t, "API_VALIDATION_ERROR", pe.ErrorCode)
}

func TestDisburse_invalidAmountIsPermanentError(t *testing.T) {
	c := NewClient(Config{})

	req := disbursementReq("11111111", "dsb-1")
	req.Amount = 0

	_, err := c.Disburse(context.Background(), req)

	require.Error(t, err)
	require.True(t, pg_provider.IsPermanent(err))
}
