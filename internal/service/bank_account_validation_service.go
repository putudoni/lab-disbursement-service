package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"lab-disbursement-service/internal/model"
	"lab-disbursement-service/pkg/pg_provider"
)

type bankAccountRepository interface {
	FindPendingValidation() ([]model.BankAccount, error)
	UpdateValidationStatus(id int64, status string, validationID string) error
	FindByID(id int64) (*model.BankAccount, error)
}

type auditTrailRepository interface {
	Append(ctx context.Context, entry *model.AuditLog) error
}

const auditActorSystem = "system"

type BankAccountValidationService struct {
	repo      bankAccountRepository
	validator pg_provider.BankAccountValidator
	audit     auditTrailRepository
}

func NewBankAccountValidationService(
	repo bankAccountRepository,
	validator pg_provider.BankAccountValidator,
	audit auditTrailRepository,
) *BankAccountValidationService {
	return &BankAccountValidationService{
		repo:      repo,
		validator: validator,
		audit:     audit,
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

func (s *BankAccountValidationService) RecordView(ctx context.Context, acc model.BankAccount, actor string, correlationID string) {
	s.writeAudit(ctx, acc, model.AuditActionViewed, acc.ValidationStatus, actor, correlationID, nil)
}

func (s *BankAccountValidationService) validateAccount(ctx context.Context, acc model.BankAccount) {
	req := pg_provider.BankAccountValidationRequest{
		BankAccountNumber: acc.AccountNumber,
		BankCode:          acc.BankCode,
		AccountHolderName: acc.AccountHolderName,
	}

	resp, err := s.validator.Validate(ctx, req)

	var (
		status       string
		validationID string
		action       string
	)

	switch {
	case err != nil && pg_provider.IsTransient(err):
		status, action = model.ValidationStatusError, model.AuditActionValidationError
	case err != nil:
		status, action = model.ValidationStatusInvalid, model.AuditActionInvalid
	case resp.Status == "verified":
		status, validationID, action = model.ValidationStatusValidated, resp.ID, model.AuditActionValidated
	default:
		status, validationID, action = model.ValidationStatusInvalid, resp.ID, model.AuditActionInvalid
	}

	_ = s.repo.UpdateValidationStatus(acc.ID, status, validationID)
	s.writeAudit(ctx, acc, action, status, auditActorSystem, "", map[string]string{
		"to_status":     status,
		"validation_id": validationID,
	})
}

func (s *BankAccountValidationService) writeAudit(
	ctx context.Context,
	acc model.BankAccount,
	action string,
	status string,
	actor string,
	correlationID string,
	extra map[string]string,
) {
	if s.audit == nil {
		return
	}

	payload := map[string]string{
		"bank_code":      acc.BankCode,
		"account_number": maskAccountNumber(acc.AccountNumber),
		"from_status":    acc.ValidationStatus,
		"to_status":      status,
	}
	for k, v := range extra {
		payload[k] = v
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	_ = s.audit.Append(ctx, &model.AuditLog{
		EntityType:    model.AuditEntityBankAccount,
		EntityID:      acc.ID,
		Action:        action,
		Actor:         actor,
		CorrelationID: correlationID,
		Payload:       string(data),
		CreatedAt:     time.Now().UTC(),
	})
}

func maskAccountNumber(accountNumber string) string {
	if len(accountNumber) <= 4 {
		return strings.Repeat("*", len(accountNumber))
	}

	return strings.Repeat("*", len(accountNumber)-4) + accountNumber[len(accountNumber)-4:]
}
