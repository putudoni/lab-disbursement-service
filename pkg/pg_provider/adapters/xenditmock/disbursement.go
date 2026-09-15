package xenditmock

import (
	"context"

	"lab-disbursement-service/pkg/pg_provider"
)

func (c *Client) Disburse(ctx context.Context, req pg_provider.DisbursementRequest) (*pg_provider.DisbursementResponse, error) {
	if req.ExternalID == "" || req.BankCode == "" || req.AccountNumber == "" || req.Description == "" || req.Amount <= 0 {
		return nil, &pg_provider.ProviderError{
			ErrorCode: "API_VALIDATION_ERROR",
			Message:   "external_id, bank_code, account_number, description and amount are required",
		}
	}

	resp := &pg_provider.DisbursementResponse{
		ID:          "dsb-trx-" + req.ExternalID,
		ExternalID:  req.ExternalID,
		Amount:      req.Amount,
		FailureCode: "",
	}

	switch req.AccountNumber {
	case "11111111":
		resp.Status = pg_provider.DisbursementStatusCompleted
	case "22222222":
		c.mu.Lock()
		c.submits[req.ExternalID]++
		attempt := c.submits[req.ExternalID]
		c.mu.Unlock()

		if attempt == 1 {
			resp.Status = pg_provider.DisbursementStatusPending
		} else {
			resp.Status = pg_provider.DisbursementStatusCompleted
		}
	default:
		resp.Status = pg_provider.DisbursementStatusFailed
		resp.FailureCode = "INVALID_DESTINATION"
	}

	return resp, nil
}