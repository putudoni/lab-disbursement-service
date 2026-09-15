package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"lab-disbursement-service/internal/model"
	"lab-disbursement-service/pkg/pg_provider"
)

type disbursementStatusUpdate struct {
	id         int64
	status     string
	providerID string
}

type fakeDisbursementRepository struct {
	pendingOrProcessing []model.Disbursement
	fetchErr            error

	bankByID    map[int64]*model.BankAccount
	bankByIDErr error

	updates []disbursementStatusUpdate
	byID    map[int64]*model.Disbursement
	byIDErr error
}

func (f *fakeDisbursementRepository) FindPendingOrProcessing() ([]model.Disbursement, error) {
	return f.pendingOrProcessing, f.fetchErr
}

func (f *fakeDisbursementRepository) FindBankAccountByID(id int64) (*model.BankAccount, error) {
	if f.bankByIDErr != nil {
		return nil, f.bankByIDErr
	}
	return f.bankByID[id], nil
}

func (f *fakeDisbursementRepository) UpdateDisbursementStatus(id int64, status string, providerID string) error {
	f.updates = append(f.updates, disbursementStatusUpdate{id: id, status: status, providerID: providerID})
	return nil
}

func (f *fakeDisbursementRepository) FindByID(id int64) (*model.Disbursement, error) {
	if f.byIDErr != nil {
		return nil, f.byIDErr
	}
	return f.byID[id], nil
}

type fakeDisbursementProvider struct {
	resp  *pg_provider.DisbursementResponse
	err   error
	calls int
	last  pg_provider.DisbursementRequest
}

func (f *fakeDisbursementProvider) Disburse(ctx context.Context, req pg_provider.DisbursementRequest) (*pg_provider.DisbursementResponse, error) {
	f.calls++
	f.last = req
	return f.resp, f.err
}

func makeDisbursement(id int64, status string, accountNumber string) model.Disbursement {
	return model.Disbursement{
		ID:                 id,
		LoanID:             id * 10,
		BankAccountID:      id,
		BankCode:           "BCA",
		AccountNumber:      accountNumber,
		AccountHolderName:  "Budi",
		Description:        "Pencairan pinjaman",
		Amount:             500000,
		DisbursementStatus: status,
	}
}

func validatedBankAccount(id int64) *model.BankAccount {
	return &model.BankAccount{
		ID:                id,
		AccountNumber:     "11111111",
		BankCode:          "BCA",
		AccountHolderName: "Budi",
		ValidationStatus:  model.ValidationStatusValidated,
	}
}

func TestProcessPendingDisbursements_validatedBankAccountBecomesSuccess(t *testing.T) {
	repo := &fakeDisbursementRepository{
		pendingOrProcessing: []model.Disbursement{makeDisbursement(1, model.DisbursementStatusPending, "11111111")},
		bankByID:            map[int64]*model.BankAccount{1: validatedBankAccount(1)},
	}
	disburser := &fakeDisbursementProvider{
		resp: &pg_provider.DisbursementResponse{ID: "trx-1", ExternalID: "dsb-1", Status: pg_provider.DisbursementStatusCompleted, Amount: 500000},
	}
	svc := NewDisbursementService(repo, disburser, nil)

	err := svc.ProcessPendingDisbursements(context.Background())

	require.NoError(t, err)
	require.Equal(t, 1, disburser.calls)
	require.Equal(t, "dsb-1", disburser.last.ExternalID)
	require.Equal(t, "11111111", disburser.last.AccountNumber)
	require.Equal(t, "Pencairan pinjaman", disburser.last.Description)
	require.Equal(t, int64(500000), disburser.last.Amount)
	require.Equal(t, []disbursementStatusUpdate{{id: 1, status: model.DisbursementStatusSuccess, providerID: "trx-1"}}, repo.updates)
}

func TestProcessPendingDisbursements_unvalidatedBankAccountFails(t *testing.T) {
	account := validatedBankAccount(1)
	account.ValidationStatus = model.ValidationStatusPending

	repo := &fakeDisbursementRepository{
		pendingOrProcessing: []model.Disbursement{makeDisbursement(1, model.DisbursementStatusPending, "11111111")},
		bankByID:            map[int64]*model.BankAccount{1: account},
	}
	disburser := &fakeDisbursementProvider{}
	svc := NewDisbursementService(repo, disburser, nil)

	err := svc.ProcessPendingDisbursements(context.Background())

	require.NoError(t, err)
	require.Equal(t, 0, disburser.calls)
	require.Equal(t, []disbursementStatusUpdate{{id: 1, status: model.DisbursementStatusFailed, providerID: ""}}, repo.updates)
}

func TestProcessPendingDisbursements_missingBankAccountFails(t *testing.T) {
	repo := &fakeDisbursementRepository{
		pendingOrProcessing: []model.Disbursement{makeDisbursement(1, model.DisbursementStatusPending, "11111111")},
		bankByID:            map[int64]*model.BankAccount{},
	}
	disburser := &fakeDisbursementProvider{}
	svc := NewDisbursementService(repo, disburser, nil)

	err := svc.ProcessPendingDisbursements(context.Background())

	require.NoError(t, err)
	require.Equal(t, 0, disburser.calls)
	require.Equal(t, []disbursementStatusUpdate{{id: 1, status: model.DisbursementStatusFailed, providerID: ""}}, repo.updates)
}

func TestProcessPendingDisbursements_transientErrorIsTerminal(t *testing.T) {
	repo := &fakeDisbursementRepository{
		pendingOrProcessing: []model.Disbursement{makeDisbursement(1, model.DisbursementStatusPending, "11111111")},
		bankByID:            map[int64]*model.BankAccount{1: validatedBankAccount(1)},
	}
	disburser := &fakeDisbursementProvider{err: errors.New("provider unavailable")}
	svc := NewDisbursementService(repo, disburser, nil)

	err := svc.ProcessPendingDisbursements(context.Background())

	require.NoError(t, err)
	require.Equal(t, []disbursementStatusUpdate{{id: 1, status: model.DisbursementStatusError, providerID: ""}}, repo.updates)
}

func TestProcessPendingDisbursements_permanentErrorFails(t *testing.T) {
	repo := &fakeDisbursementRepository{
		pendingOrProcessing: []model.Disbursement{makeDisbursement(1, model.DisbursementStatusPending, "11111111")},
		bankByID:            map[int64]*model.BankAccount{1: validatedBankAccount(1)},
	}
	disburser := &fakeDisbursementProvider{
		err: &pg_provider.ProviderError{StatusCode: 400, ErrorCode: "INSUFFICIENT_BALANCE", Message: "balance not enough"},
	}
	svc := NewDisbursementService(repo, disburser, nil)

	err := svc.ProcessPendingDisbursements(context.Background())

	require.NoError(t, err)
	require.Equal(t, []disbursementStatusUpdate{{id: 1, status: model.DisbursementStatusFailed, providerID: ""}}, repo.updates)
}

func TestProcessPendingDisbursements_providerFailedFails(t *testing.T) {
	repo := &fakeDisbursementRepository{
		pendingOrProcessing: []model.Disbursement{makeDisbursement(1, model.DisbursementStatusPending, "11111111")},
		bankByID:            map[int64]*model.BankAccount{1: validatedBankAccount(1)},
	}
	disburser := &fakeDisbursementProvider{
		resp: &pg_provider.DisbursementResponse{ID: "trx-1", Status: pg_provider.DisbursementStatusFailed},
	}
	svc := NewDisbursementService(repo, disburser, nil)

	err := svc.ProcessPendingDisbursements(context.Background())

	require.NoError(t, err)
	require.Equal(t, []disbursementStatusUpdate{{id: 1, status: model.DisbursementStatusFailed, providerID: "trx-1"}}, repo.updates)
}

func TestProcessPendingDisbursements_providerPendingMovesToProcessingAndRetries(t *testing.T) {
	repo := &fakeDisbursementRepository{
		pendingOrProcessing: []model.Disbursement{makeDisbursement(1, model.DisbursementStatusPending, "11111111")},
		bankByID:            map[int64]*model.BankAccount{1: validatedBankAccount(1)},
	}
	disburser := &fakeDisbursementProvider{
		resp: &pg_provider.DisbursementResponse{ID: "trx-1", Status: pg_provider.DisbursementStatusPending},
	}
	svc := NewDisbursementService(repo, disburser, nil)

	err := svc.ProcessPendingDisbursements(context.Background())
	require.NoError(t, err)
	require.Equal(t, []disbursementStatusUpdate{{id: 1, status: model.DisbursementStatusProcessing, providerID: "trx-1"}}, repo.updates)

	repo.pendingOrProcessing = []model.Disbursement{
		func() model.Disbursement {
			d := makeDisbursement(1, model.DisbursementStatusProcessing, "11111111")
			d.ProviderID = "trx-1"
			return d
		}(),
	}
	disburser.resp = &pg_provider.DisbursementResponse{ID: "trx-1", Status: pg_provider.DisbursementStatusCompleted}

	err = svc.ProcessPendingDisbursements(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, disburser.calls)
	require.Equal(t, []disbursementStatusUpdate{
		{id: 1, status: model.DisbursementStatusProcessing, providerID: "trx-1"},
		{id: 1, status: model.DisbursementStatusSuccess, providerID: "trx-1"},
	}, repo.updates)
}

func TestProcessPendingDisbursements_processingDoesNotRecheckBankAccount(t *testing.T) {
	repo := &fakeDisbursementRepository{
		pendingOrProcessing: []model.Disbursement{
			func() model.Disbursement {
				d := makeDisbursement(1, model.DisbursementStatusProcessing, "11111111")
				d.ProviderID = "trx-1"
				return d
			}(),
		},
	}
	disburser := &fakeDisbursementProvider{
		resp: &pg_provider.DisbursementResponse{ID: "trx-1", Status: pg_provider.DisbursementStatusCompleted},
	}
	svc := NewDisbursementService(repo, disburser, nil)

	err := svc.ProcessPendingDisbursements(context.Background())

	require.NoError(t, err)
	require.Equal(t, 1, disburser.calls)
	require.Equal(t, []disbursementStatusUpdate{{id: 1, status: model.DisbursementStatusSuccess, providerID: "trx-1"}}, repo.updates)
}

func TestProcessPendingDisbursements_repoErrorIsReturned(t *testing.T) {
	repo := &fakeDisbursementRepository{fetchErr: errors.New("db down")}
	disburser := &fakeDisbursementProvider{
		resp: &pg_provider.DisbursementResponse{Status: pg_provider.DisbursementStatusCompleted},
	}
	svc := NewDisbursementService(repo, disburser, nil)

	err := svc.ProcessPendingDisbursements(context.Background())

	require.Error(t, err)
	require.Equal(t, 0, disburser.calls)
}

func TestGetDisbursement_found(t *testing.T) {
	repo := &fakeDisbursementRepository{
		byID: map[int64]*model.Disbursement{1: func() *model.Disbursement {
			d := makeDisbursement(1, model.DisbursementStatusPending, "11111111")
			return &d
		}()},
	}
	svc := NewDisbursementService(repo, &fakeDisbursementProvider{}, nil)

	disbursement, err := svc.GetDisbursement(context.Background(), 1)

	require.NoError(t, err)
	require.NotNil(t, disbursement)
	require.Equal(t, int64(1), disbursement.ID)
}

func TestGetDisbursement_notFound(t *testing.T) {
	repo := &fakeDisbursementRepository{byID: map[int64]*model.Disbursement{}}
	svc := NewDisbursementService(repo, &fakeDisbursementProvider{}, nil)

	disbursement, err := svc.GetDisbursement(context.Background(), 1)

	require.NoError(t, err)
	require.Nil(t, disbursement)
}

func TestDisbursementService_nilAuditIsSafe(t *testing.T) {
	repo := &fakeDisbursementRepository{
		pendingOrProcessing: []model.Disbursement{makeDisbursement(1, model.DisbursementStatusPending, "11111111")},
		bankByID:            map[int64]*model.BankAccount{1: validatedBankAccount(1)},
	}
	disburser := &fakeDisbursementProvider{
		resp: &pg_provider.DisbursementResponse{Status: pg_provider.DisbursementStatusCompleted},
	}
	svc := NewDisbursementService(repo, disburser, nil)

	require.NotPanics(t, func() {
		_ = svc.ProcessPendingDisbursements(context.Background())
	})
}

func TestProcessPendingDisbursements_writesAuditTrailWithMaskedPII(t *testing.T) {
	auditRepo := &fakeAuditTrailRepository{}
	repo := &fakeDisbursementRepository{
		pendingOrProcessing: []model.Disbursement{makeDisbursement(1, model.DisbursementStatusPending, "11111111")},
		bankByID:            map[int64]*model.BankAccount{1: validatedBankAccount(1)},
	}
	disburser := &fakeDisbursementProvider{
		resp: &pg_provider.DisbursementResponse{ID: "trx-1", ExternalID: "dsb-1", Status: pg_provider.DisbursementStatusCompleted},
	}
	svc := NewDisbursementService(repo, disburser, auditRepo)

	err := svc.ProcessPendingDisbursements(context.Background())

	require.NoError(t, err)
	require.Len(t, auditRepo.entries, 1)

	entry := auditRepo.entries[0]
	require.Equal(t, model.AuditEntityDisbursement, entry.EntityType)
	require.Equal(t, int64(1), entry.EntityID)
	require.Equal(t, model.AuditActionDisbursementSuccess, entry.Action)
	require.Equal(t, auditActorSystem, entry.Actor)
	require.Empty(t, entry.CorrelationID)
	require.NotContains(t, entry.Payload, "11111111")
	require.NotContains(t, entry.Payload, "Budi")

	var payload map[string]string
	require.NoError(t, json.Unmarshal([]byte(entry.Payload), &payload))
	require.Equal(t, "****1111", payload["account_number"])
	require.Equal(t, "500000", payload["amount"])
	require.Equal(t, "dsb-1", payload["external_id"])
	require.Equal(t, model.DisbursementStatusPending, payload["from_status"])
	require.Equal(t, model.DisbursementStatusSuccess, payload["to_status"])
	require.Equal(t, "trx-1", payload["provider_id"])
}

func TestProcessPendingDisbursements_auditReasonFromProviderFailure(t *testing.T) {
	auditRepo := &fakeAuditTrailRepository{}
	repo := &fakeDisbursementRepository{
		pendingOrProcessing: []model.Disbursement{makeDisbursement(1, model.DisbursementStatusPending, "11111111")},
		bankByID:            map[int64]*model.BankAccount{1: validatedBankAccount(1)},
	}
	disburser := &fakeDisbursementProvider{
		resp: &pg_provider.DisbursementResponse{ID: "trx-1", Status: pg_provider.DisbursementStatusFailed, FailureCode: "INVALID_DESTINATION"},
	}
	svc := NewDisbursementService(repo, disburser, auditRepo)

	err := svc.ProcessPendingDisbursements(context.Background())

	require.NoError(t, err)
	require.Len(t, auditRepo.entries, 1)
	require.Equal(t, model.AuditActionDisbursementFailed, auditRepo.entries[0].Action)

	var payload map[string]string
	require.NoError(t, json.Unmarshal([]byte(auditRepo.entries[0].Payload), &payload))
	require.Equal(t, model.DisbursementStatusFailed, payload["to_status"])
	require.Equal(t, "INVALID_DESTINATION", payload["reason"])
}

func TestProcessPendingDisbursements_writesAuditTrailOnTransientError(t *testing.T) {
	auditRepo := &fakeAuditTrailRepository{}
	repo := &fakeDisbursementRepository{
		pendingOrProcessing: []model.Disbursement{makeDisbursement(1, model.DisbursementStatusPending, "11111111")},
		bankByID:            map[int64]*model.BankAccount{1: validatedBankAccount(1)},
	}
	disburser := &fakeDisbursementProvider{err: errors.New("provider unavailable")}
	svc := NewDisbursementService(repo, disburser, auditRepo)

	err := svc.ProcessPendingDisbursements(context.Background())

	require.NoError(t, err)
	require.Len(t, auditRepo.entries, 1)
	require.Equal(t, model.AuditActionDisbursementError, auditRepo.entries[0].Action)

	var payload map[string]string
	require.NoError(t, json.Unmarshal([]byte(auditRepo.entries[0].Payload), &payload))
	require.Equal(t, model.DisbursementStatusError, payload["to_status"])
}

func TestGetDisbursement_writesAuditTrailOnView(t *testing.T) {
	auditRepo := &fakeAuditTrailRepository{}
	repo := &fakeDisbursementRepository{
		byID: map[int64]*model.Disbursement{1: func() *model.Disbursement {
			d := makeDisbursement(1, model.DisbursementStatusPending, "1234567890")
			return &d
		}()},
	}
	svc := NewDisbursementService(repo, &fakeDisbursementProvider{}, auditRepo)

	disbursement, err := svc.GetDisbursement(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, disbursement)

	svc.RecordView(context.Background(), *disbursement, "api:http://localhost:8080", "corr-1")

	require.Len(t, auditRepo.entries, 1)

	entry := auditRepo.entries[0]
	require.Equal(t, model.AuditActionDisbursementViewed, entry.Action)
	require.Equal(t, "api:http://localhost:8080", entry.Actor)
	require.Equal(t, "corr-1", entry.CorrelationID)

	var payload map[string]string
	require.NoError(t, json.Unmarshal([]byte(entry.Payload), &payload))
	require.Equal(t, "******7890", payload["account_number"])
	require.NotContains(t, entry.Payload, "1234567890")
}
