package xendit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/cenkalti/backoff/v5"

	"lab-disbursement-service/pkg/pg_provider"
)

func (c *Client) Validate(ctx context.Context, req pg_provider.BankAccountValidationRequest) (*pg_provider.BankAccountValidationResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, pg_provider.NewLocalError("REQUEST_ERROR", fmt.Sprintf("marshal request body: %v", err))
	}

	body, statusCode, err := c.doRequest(ctx, payload)
	if err != nil {
		return nil, err
	}

	if statusCode >= 200 && statusCode < 300 {
		var resp bankAccountValidationResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, pg_provider.NewLocalError("RESPONSE_ERROR", fmt.Sprintf("unmarshal response: %v", err))
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

	return nil, c.parseError(body, statusCode)
}

type requestResult struct {
	body       []byte
	statusCode int
}

func (c *Client) doRequest(ctx context.Context, body []byte) ([]byte, int, error) {
	headers := map[string]string{
		"Authorization": c.basicAuth(),
	}

	var permanent *pg_provider.ProviderError

	result, err := backoff.Retry(ctx, func() (requestResult, error) {
		resp, err := c.client.PostWithHeaders(ctx, requestPath, nil, body, headers)
		if err != nil {
			return requestResult{}, err
		}

		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return requestResult{}, err
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return requestResult{body: data, statusCode: resp.StatusCode}, nil
		}

		if resp.StatusCode >= 500 {
			return requestResult{}, &pg_provider.ProviderError{
				StatusCode: resp.StatusCode,
				ErrorCode:  "PROVIDER_UNAVAILABLE",
				Message:    string(data),
			}
		}

		permanent = c.parseError(data, resp.StatusCode)
		return requestResult{}, nil
	},
		backoff.WithBackOff(c.backoff()),
		backoff.WithMaxTries(uint(c.config.RetryMax)+1),
	)

	if permanent != nil {
		return nil, permanent.StatusCode, permanent
	}
	if err != nil {
		return nil, 0, err
	}

	return result.body, result.statusCode, nil
}

type bankAccountValidationResponse struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	NameMatchResult   string `json:"name_match_result"`
	AccountNumber     string `json:"account_number"`
	BankCode          string `json:"bank_code"`
	AccountHolderName string `json:"name"`
}

func (c *Client) parseError(data []byte, statusCode int) *pg_provider.ProviderError {
	var e providerErrorBody
	if err := json.Unmarshal(data, &e); err == nil && e.Message != "" {
		return &pg_provider.ProviderError{
			StatusCode: statusCode,
			ErrorCode:  e.ErrorCode,
			Message:    e.Message,
		}
	}

	return &pg_provider.ProviderError{
		StatusCode: statusCode,
		ErrorCode:  "UNKNOWN_ERROR",
		Message:    fmt.Sprintf("unexpected status code %d: %s", statusCode, string(data)),
	}
}

type providerErrorBody struct {
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}
