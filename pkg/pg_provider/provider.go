package pg_provider

import "context"

type BankAccountValidator interface {
	Validate(ctx context.Context, req BankAccountValidationRequest) (*BankAccountValidationResponse, error)
}
