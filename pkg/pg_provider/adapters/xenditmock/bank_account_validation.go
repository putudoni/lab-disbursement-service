package xenditmock

import (
	"context"
	"fmt"
	"time"

	"lab-disbursement-service/pkg/pg_provider"
)

type Client struct {
	config Config
}

func NewClient(cfg Config) *Client {
	return &Client{config: cfg}
}

func (c *Client) Validate(ctx context.Context, req pg_provider.BankAccountValidationRequest) (*pg_provider.BankAccountValidationResponse, error) {
	if req.BankAccountNumber == "" || req.BankCode == "" {
		return nil, &pg_provider.ProviderError{
			ErrorCode: "API_VALIDATION_ERROR",
			Message:   "bank_account_number and bank_code are required",
		}
	}

	resp := &pg_provider.BankAccountValidationResponse{
		ID:                fmt.Sprintf("val-%d", time.Now().UnixNano()),
		BankCode:          req.BankCode,
		AccountNumber:     req.BankAccountNumber,
		AccountHolderName: req.AccountHolderName,
	}

	switch req.BankAccountNumber {
	case "11111111":
		resp.Status = "verified"
		resp.NameMatchResult = "match"
	case "22222222":
		resp.Status = "verified"
		resp.NameMatchResult = "mismatch"
	default:
		resp.Status = "unverified"
		resp.NameMatchResult = "not_found"
	}

	return resp, nil
}
