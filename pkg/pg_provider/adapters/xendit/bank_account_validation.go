package xendit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"lab-disbursement-service/pkg/pg_provider"
)

func (c *Client) Validate(ctx context.Context, req pg_provider.BankAccountValidationRequest) (*pg_provider.BankAccountValidationResponse, error) {
	data, statusCode, err := c.doRequest(ctx, http.MethodPost, "/identity/v2/bank_account_validation", req)
	if err != nil {
		return nil, err
	}

	if statusCode >= 200 && statusCode < 300 {
		var resp bankAccountValidationResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("pg_provider/xendit: unmarshal response: %w", err)
		}

		return &pg_provider.BankAccountValidationResponse{
			ID:                resp.ID,
			Status:            resp.Status,
			NameMatchResult:   resp.NameMatchResult,
			AccountNumber:     resp.AccountNumber,
			BankCode:          resp.BankCode,
			AccountHolderName: resp.AccountHolderName,
		}, nil
	}

	return nil, c.parseError(data, statusCode)
}

type bankAccountValidationResponse struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	NameMatchResult   string `json:"name_match_result"`
	AccountNumber     string `json:"account_number"`
	BankCode          string `json:"bank_code"`
	AccountHolderName string `json:"name"`
}

func (c *Client) parseError(data []byte, statusCode int) error {
	var e providerErrorBody
	if err := json.Unmarshal(data, &e); err == nil && e.Message != "" {
		return &pg_provider.ProviderError{
			ErrorCode: e.ErrorCode,
			Message:   e.Message,
		}
	}

	return fmt.Errorf("pg_provider/xendit: unexpected status code %d: %s", statusCode, string(data))
}

type providerErrorBody struct {
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}