package service

import (
	"context"

	"lab-disbursement-service/internal/model"
	"lab-disbursement-service/internal/repository"
	"lab-disbursement-service/pkg/pg_provider"
)

type BankAccountValidationService struct {
	repo      *repository.BankAccountRepository
	validator pg_provider.BankAccountValidator
}

func NewBankAccountValidationService(
	repo *repository.BankAccountRepository,
	validator pg_provider.BankAccountValidator,
) *BankAccountValidationService {
	return &BankAccountValidationService{
		repo:      repo,
		validator: validator,
	}
}

func (s *BankAccountValidationService) ProcessPendingValidations(ctx context.Context) error {
	accounts, err := s.repo.FindPendingValidation()
	if err != nil {
		return err
	}

	for _, acc := range accounts {
		s.validateAccount(ctx, acc)
	}

	return nil
}

func (s *BankAccountValidationService) GetBankAccount(ctx context.Context, id int64) (*model.BankAccount, error) {
	return s.repo.FindByID(id)
}

func (s *BankAccountValidationService) validateAccount(ctx context.Context, acc model.BankAccount) {
	req := pg_provider.BankAccountValidationRequest{
		BankAccountNumber: acc.AccountNumber,
		BankCode:          acc.BankCode,
		AccountHolderName: acc.AccountHolderName,
	}

	resp, err := s.validator.Validate(ctx, req)
	if err != nil {
		_ = s.repo.UpdateValidationStatus(acc.ID, model.ValidationStatusFailed, "")
		return
	}

	switch resp.Status {
	case "verified":
		_ = s.repo.UpdateValidationStatus(acc.ID, model.ValidationStatusValidated, resp.ID)
	default:
		_ = s.repo.UpdateValidationStatus(acc.ID, model.ValidationStatusFailed, resp.ID)
	}
}
