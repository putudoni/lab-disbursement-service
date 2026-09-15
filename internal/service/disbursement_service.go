package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"lab-disbursement-service/internal/model"
	"lab-disbursement-service/pkg/pg_provider"
)

type disbursementRepository interface {
	FindPendingOrProcessing() ([]model.Disbursement, error)
	FindBankAccountByID(id int64) (*model.BankAccount, error)
	UpdateDisbursementStatus(id int64, status string, providerID string) error
	FindByID(id int64) (*model.Disbursement, error)
}

type DisbursementService struct {
	repo      disbursementRepository
	disburser pg_provider.DisbursementProvider
	audit     auditTrailRepository
}

func NewDisbursementService(
	repo disbursementRepository,
	disburser pg_provider.DisbursementProvider,
	audit auditTrailRepository,
) *DisbursementService {
	return &DisbursementService{
		repo:      repo,
		disburser: disburser,
		audit:     audit,
	}
}

func (s *DisbursementService) ProcessPendingDisbursements(ctx context.Context) error {
	disbursements, err := s.repo.FindPendingOrProcessing()
	if err != nil {
		return err
	}

	for _, d := range disbursements {
		if d.DisbursementStatus != model.DisbursementStatusPending &&
			d.DisbursementStatus != model.DisbursementStatusProcessing {
			continue
		}

		s.processDisbursement(ctx, d)
	}

	return nil
}

func (s *DisbursementService) GetDisbursement(ctx context.Context, id int64) (*model.Disbursement, error) {
	return s.repo.FindByID(id)
}

func (s *DisbursementService) RecordView(ctx context.Context, d model.Disbursement, actor string, correlationID string) {
	s.writeAudit(ctx, d, model.AuditActionDisbursementViewed, d.DisbursementStatus, actor, correlationID, nil)
}

func (s *DisbursementService) processDisbursement(ctx context.Context, d model.Disbursement) {
	if d.DisbursementStatus == model.DisbursementStatusPending {
		account, err := s.repo.FindBankAccountByID(d.BankAccountID)
		if err != nil {
			s.writeAudit(ctx, d, model.AuditActionDisbursementError, model.DisbursementStatusError,
				auditActorSystem, "", map[string]string{"reason": "bank_account_lookup_failed"})
			return
		}

		if account == nil {
			s.failWithReason(ctx, d, "bank_account_not_found")
			return
		}
		if account.ValidationStatus != model.ValidationStatusValidated {
			s.failWithReason(ctx, d, "bank_account_not_validated")
			return
		}
	}

	req := pg_provider.DisbursementRequest{
		ExternalID:        fmt.Sprintf("dsb-%d", d.ID),
		BankCode:          d.BankCode,
		AccountHolderName: d.AccountHolderName,
		AccountNumber:     d.AccountNumber,
		Description:       d.Description,
		Amount:            d.Amount,
	}

	resp, err := s.disburser.Disburse(ctx, req)

	var (
		status     string
		providerID string
		action     string
		reason     string
	)

	switch {
	case err != nil && pg_provider.IsTransient(err):
		status, action = model.DisbursementStatusError, model.AuditActionDisbursementError
	case err != nil:
		status, action = model.DisbursementStatusFailed, model.AuditActionDisbursementFailed
	case resp.Status == pg_provider.DisbursementStatusCompleted:
		status, providerID, action = model.DisbursementStatusSuccess, resp.ID, model.AuditActionDisbursementSuccess
	case resp.Status == pg_provider.DisbursementStatusFailed:
		status, providerID, action = model.DisbursementStatusFailed, resp.ID, model.AuditActionDisbursementFailed
		reason = resp.FailureCode
	default:
		status, providerID, action = model.DisbursementStatusProcessing, resp.ID, model.AuditActionDisbursementProcessing
	}

	_ = s.repo.UpdateDisbursementStatus(d.ID, status, providerID)

	extras := map[string]string{
		"to_status":   status,
		"provider_id": providerID,
	}
	if reason != "" {
		extras["reason"] = reason
	}

	s.writeAudit(ctx, d, action, status, auditActorSystem, "", extras)
}

func (s *DisbursementService) failWithReason(ctx context.Context, d model.Disbursement, reason string) {
	_ = s.repo.UpdateDisbursementStatus(d.ID, model.DisbursementStatusFailed, "")
	s.writeAudit(ctx, d, model.AuditActionDisbursementFailed, model.DisbursementStatusFailed,
		auditActorSystem, "", map[string]string{
			"to_status": model.DisbursementStatusFailed,
			"reason":    reason,
		})
}

func (s *DisbursementService) writeAudit(
	ctx context.Context,
	d model.Disbursement,
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
		"bank_code":      d.BankCode,
		"account_number": model.MaskAccountNumber(d.AccountNumber),
		"amount":         strconv.FormatInt(d.Amount, 10),
		"external_id":    fmt.Sprintf("dsb-%d", d.ID),
		"from_status":    d.DisbursementStatus,
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
		EntityType:    model.AuditEntityDisbursement,
		EntityID:      d.ID,
		Action:        action,
		Actor:         actor,
		CorrelationID: correlationID,
		Payload:       string(data),
		CreatedAt:     time.Now().UTC(),
	})
}
