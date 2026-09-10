package service

import (
	"context"

	"lab-disbursement-service/internal/model"
	"lab-disbursement-service/pkg/pg_provider"
)

type bankAccountRepository interface {
	FindPendingValidation() ([]model.BankAccount, error)
	UpdateValidationStatus(id int64, status string, validationID string) error
	FindByID(id int64) (*model.BankAccount, error)
}

type BankAccountValidationService struct {
	repo      bankAccountRepository
	validator pg_provider.BankAccountValidator
}

func NewBankAccountValidationService(
	repo bankAccountRepository,
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
		if acc.ValidationStatus != model.ValidationStatusPending {
			continue
		}

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
		_ = s.repo.UpdateValidationStatus(acc.ID, model.ValidationStatusInvalid, "")
		return
	}

	if resp.Status == "verified" {
		_ = s.repo.UpdateValidationStatus(acc.ID, model.ValidationStatusValidated, resp.ID)
		return
	}

	_ = s.repo.UpdateValidationStatus(acc.ID, model.ValidationStatusInvalid, resp.ID)
}