package pg_provider

import "context"

type BankAccountValidator interface {
	Validate(ctx context.Context, req BankAccountValidationRequest) (*BankAccountValidationResponse, error)
}

type DisbursementProvider interface {
	Disburse(ctx context.Context, req DisbursementRequest) (*DisbursementResponse, error)
}
