package xendit

import (
	"context"
	"encoding/json"
	"fmt"

	"lab-disbursement-service/pkg/pg_provider"
)

func (c *Client) Disburse(ctx context.Context, req pg_provider.DisbursementRequest) (*pg_provider.DisbursementResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, pg_provider.NewLocalError("REQUEST_ERROR", fmt.Sprintf("marshal request body: %v", err))
	}

	extraHeaders := map[string]string{"X-Idempotency-Key": req.ExternalID}
	body, statusCode, err := c.doRequest(ctx, disbursementPath, payload, extraHeaders)
	if err != nil {
		return nil, err
	}

	if statusCode >= 200 && statusCode < 300 {
		var resp disbursementResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, pg_provider.NewLocalError("RESPONSE_ERROR", fmt.Sprintf("unmarshal response: %v", err))
		}

		return &pg_provider.DisbursementResponse{
			ID:          resp.ID,
			ExternalID:  resp.ExternalID,
			Status:      resp.Status,
			Amount:      resp.Amount,
			FailureCode: resp.FailureCode,
		}, nil
	}

	return nil, c.parseError(body, statusCode)
}

type disbursementResponse struct {
	ID          string `json:"id"`
	ExternalID  string `json:"external_id"`
	Status      string `json:"status"`
	Amount      int64  `json:"amount"`
	FailureCode string `json:"failure_code"`
}
