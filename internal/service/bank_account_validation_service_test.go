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

type fakeAuditTrailRepository struct {
	entries []*model.AuditLog
	err     error
}

func (f *fakeAuditTrailRepository) Append(ctx context.Context, entry *model.AuditLog) error {
	if f.err != nil {
		return f.err
	}
	f.entries = append(f.entries, entry)
	return nil
}

type validationStatusUpdate struct {
	id           int64
	status       string
	validationID string
}

type fakeBankAccountRepository struct {
	pending    []model.BankAccount
	pendingErr error

	byID    map[int64]*model.BankAccount
	byIDErr error

	updates []validationStatusUpdate
}

func (f *fakeBankAccountRepository) FindPendingValidation() ([]model.BankAccount, error) {
	return f.pending, f.pendingErr
}

func (f *fakeBankAccountRepository) UpdateValidationStatus(id int64, status string, validationID string) error {
	f.updates = append(f.updates, validationStatusUpdate{id: id, status: status, validationID: validationID})
	return nil
}

func (f *fakeBankAccountRepository) FindByID(id int64) (*model.BankAccount, error) {
	if f.byIDErr != nil {
		return nil, f.byIDErr
	}
	return f.byID[id], nil
}

type fakeBankAccountValidator struct {
	resp  *pg_provider.BankAccountValidationResponse
	err   error
	calls int
	last  pg_provider.BankAccountValidationRequest
}

func (f *fakeBankAccountValidator) Validate(ctx context.Context, req pg_provider.BankAccountValidationRequest) (*pg_provider.BankAccountValidationResponse, error) {
	f.calls++
	f.last = req
	return f.resp, f.err
}

func pendingAccount(id int64, accountNumber string) model.BankAccount {
	return model.BankAccount{
		ID:                id,
		LoanID:            id * 10,
		BankCode:          "BCA",
		AccountNumber:     accountNumber,
		AccountHolderName: "Budi",
		ValidationStatus:  model.ValidationStatusPending,
	}
}

func TestProcessPendingValidations_validAccountBecomesValidated(t *testing.T) {
	repo := &fakeBankAccountRepository{
		pending: []model.BankAccount{pendingAccount(1, "11111111")},
	}
	validator := &fakeBankAccountValidator{
		resp: &pg_provider.BankAccountValidationResponse{ID: "vid-1", Status: "verified"},
	}
	svc := NewBankAccountValidationService(repo, validator, nil)

	err := svc.ProcessPendingValidations(context.Background())

	require.NoError(t, err)
	require.Equal(t, 1, validator.calls)
	require.Equal(t, "11111111", validator.last.BankAccountNumber)
	require.Equal(t, "BCA", validator.last.BankCode)
	require.Equal(t, "Budi", validator.last.AccountHolderName)
	require.Equal(t, []validationStatusUpdate{{id: 1, status: model.ValidationStatusValidated, validationID: "vid-1"}}, repo.updates)
}

func TestProcessPendingValidations_providerErrorMarksError(t *testing.T) {
	repo := &fakeBankAccountRepository{
		pending: []model.BankAccount{pendingAccount(2, "22222222")},
	}
	validator := &fakeBankAccountValidator{err: errors.New("provider unavailable")}
	svc := NewBankAccountValidationService(repo, validator, nil)

	err := svc.ProcessPendingValidations(context.Background())

	require.NoError(t, err)
	require.Equal(t, []validationStatusUpdate{{id: 2, status: model.ValidationStatusError, validationID: ""}}, repo.updates)
}

func TestProcessPendingValidations_permanentErrorMarksInvalid(t *testing.T) {
	repo := &fakeBankAccountRepository{
		pending: []model.BankAccount{pendingAccount(4, "44444444")},
	}
	validator := &fakeBankAccountValidator{
		err: &pg_provider.ProviderError{StatusCode: 404, ErrorCode: "BANK_ACCOUNT_NOT_FOUND", Message: "account not found"},
	}
	svc := NewBankAccountValidationService(repo, validator, nil)

	err := svc.ProcessPendingValidations(context.Background())

	require.NoError(t, err)
	require.Equal(t, []validationStatusUpdate{{id: 4, status: model.ValidationStatusInvalid, validationID: ""}}, repo.updates)
}

func TestProcessPendingValidations_unverifiedAccountMarksInvalid(t *testing.T) {
	repo := &fakeBankAccountRepository{
		pending: []model.BankAccount{pendingAccount(3, "99999999")},
	}
	validator := &fakeBankAccountValidator{
		resp: &pg_provider.BankAccountValidationResponse{ID: "vid-3", Status: "unverified"},
	}
	svc := NewBankAccountValidationService(repo, validator, nil)

	err := svc.ProcessPendingValidations(context.Background())

	require.NoError(t, err)
	require.Equal(t, []validationStatusUpdate{{id: 3, status: model.ValidationStatusInvalid, validationID: "vid-3"}}, repo.updates)
}

func TestProcessPendingValidations_skipsAccountsThatAlreadyHaveFinalStatus(t *testing.T) {
	validated := pendingAccount(1, "11111111")
	validated.ValidationStatus = model.ValidationStatusValidated
	invalid := pendingAccount(2, "22222222")
	invalid.ValidationStatus = model.ValidationStatusInvalid
	failed := pendingAccount(3, "33333333")
	failed.ValidationStatus = model.ValidationStatusError

	repo := &fakeBankAccountRepository{
		pending: []model.BankAccount{validated, invalid, failed},
	}
	validator := &fakeBankAccountValidator{
		resp: &pg_provider.BankAccountValidationResponse{ID: "vid-1", Status: "verified"},
	}
	svc := NewBankAccountValidationService(repo, validator, nil)

	err := svc.ProcessPendingValidations(context.Background())

	require.NoError(t, err)
	require.Equal(t, 0, validator.calls)
	require.Empty(t, repo.updates)
}

func TestProcessPendingValidations_repoErrorIsReturned(t *testing.T) {
	repo := &fakeBankAccountRepository{pendingErr: errors.New("db down")}
	validator := &fakeBankAccountValidator{
		resp: &pg_provider.BankAccountValidationResponse{ID: "vid-1", Status: "verified"},
	}
	svc := NewBankAccountValidationService(repo, validator, nil)

	err := svc.ProcessPendingValidations(context.Background())

	require.Error(t, err)
	require.Equal(t, 0, validator.calls)
}

func TestProcessPendingValidations_continuesAfterIndividualFailure(t *testing.T) {
	repo := &fakeBankAccountRepository{
		pending: []model.BankAccount{
			pendingAccount(1, "11111111"),
			pendingAccount(2, "22222222"),
		},
	}
	validator := &fakeBankAccountValidator{
		resp: &pg_provider.BankAccountValidationResponse{ID: "vid-2", Status: "unverified"},
	}
	svc := NewBankAccountValidationService(repo, validator, nil)

	err := svc.ProcessPendingValidations(context.Background())

	require.NoError(t, err)
	require.Equal(t, 2, validator.calls)
	require.Equal(t, []validationStatusUpdate{
		{id: 1, status: model.ValidationStatusInvalid, validationID: "vid-2"},
		{id: 2, status: model.ValidationStatusInvalid, validationID: "vid-2"},
	}, repo.updates)
}

func TestGetBankAccount_found(t *testing.T) {
	repo := &fakeBankAccountRepository{
		byID: map[int64]*model.BankAccount{1: func() *model.BankAccount { a := pendingAccount(1, "11111111"); return &a }()},
	}
	svc := NewBankAccountValidationService(repo, &fakeBankAccountValidator{}, nil)

	account, err := svc.GetBankAccount(context.Background(), 1)

	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(1), account.ID)
}

func TestGetBankAccount_notFound(t *testing.T) {
	repo := &fakeBankAccountRepository{byID: map[int64]*model.BankAccount{}}
	svc := NewBankAccountValidationService(repo, &fakeBankAccountValidator{}, nil)

	account, err := svc.GetBankAccount(context.Background(), 1)

	require.NoError(t, err)
	require.Nil(t, account)
}

func TestGetBankAccount_repoError(t *testing.T) {
	repo := &fakeBankAccountRepository{byIDErr: errors.New("db down")}
	svc := NewBankAccountValidationService(repo, &fakeBankAccountValidator{}, nil)

	account, err := svc.GetBankAccount(context.Background(), 1)

	require.Error(t, err)
	require.Nil(t, account)
}

func TestProcessPendingValidations_writesAuditTrailWithMaskedPII(t *testing.T) {
	auditRepo := &fakeAuditTrailRepository{}
	repo := &fakeBankAccountRepository{
		pending: []model.BankAccount{pendingAccount(1, "11111111")},
	}
	validator := &fakeBankAccountValidator{
		resp: &pg_provider.BankAccountValidationResponse{ID: "vid-1", Status: "verified"},
	}
	svc := NewBankAccountValidationService(repo, validator, auditRepo)

	err := svc.ProcessPendingValidations(context.Background())

	require.NoError(t, err)
	require.Len(t, auditRepo.entries, 1)

	entry := auditRepo.entries[0]
	require.Equal(t, model.AuditEntityBankAccount, entry.EntityType)
	require.Equal(t, int64(1), entry.EntityID)
	require.Equal(t, model.AuditActionValidated, entry.Action)
	require.Equal(t, auditActorSystem, entry.Actor)

	var payload map[string]string
	require.NoError(t, json.Unmarshal([]byte(entry.Payload), &payload))
	require.Equal(t, "****1111", payload["account_number"])
	require.NotContains(t, entry.Payload, "11111111")
	require.NotContains(t, entry.Payload, "Budi")
	require.Equal(t, model.ValidationStatusPending, payload["from_status"])
	require.Equal(t, model.ValidationStatusValidated, payload["to_status"])
	require.Equal(t, "vid-1", payload["validation_id"])
}

func TestProcessPendingValidations_writesAuditTrailOnTransientError(t *testing.T) {
	auditRepo := &fakeAuditTrailRepository{}
	repo := &fakeBankAccountRepository{
		pending: []model.BankAccount{pendingAccount(2, "22222222")},
	}
	validator := &fakeBankAccountValidator{err: errors.New("provider unavailable")}
	svc := NewBankAccountValidationService(repo, validator, auditRepo)

	err := svc.ProcessPendingValidations(context.Background())

	require.NoError(t, err)
	require.Len(t, auditRepo.entries, 1)

	entry := auditRepo.entries[0]
	require.Equal(t, model.AuditActionValidationError, entry.Action)
	require.Equal(t, auditActorSystem, entry.Actor)
	require.Empty(t, entry.CorrelationID)

	var payload map[string]string
	require.NoError(t, json.Unmarshal([]byte(entry.Payload), &payload))
	require.Equal(t, model.ValidationStatusError, payload["to_status"])
	require.Empty(t, payload["validation_id"])
}

func TestGetBankAccount_writesAuditTrailOnView(t *testing.T) {
	auditRepo := &fakeAuditTrailRepository{}
	repo := &fakeBankAccountRepository{
		byID: map[int64]*model.BankAccount{1: func() *model.BankAccount { a := pendingAccount(1, "1234567890"); return &a }()},
	}
	svc := NewBankAccountValidationService(repo, &fakeBankAccountValidator{}, auditRepo)

	account, err := svc.GetBankAccount(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, account)

	svc.RecordView(context.Background(), *account, "api:http://localhost:8080", "corr-1")

	require.Len(t, auditRepo.entries, 1)

	entry := auditRepo.entries[0]
	require.Equal(t, model.AuditActionViewed, entry.Action)
	require.Equal(t, "api:http://localhost:8080", entry.Actor)
	require.Equal(t, "corr-1", entry.CorrelationID)

	var payload map[string]string
	require.NoError(t, json.Unmarshal([]byte(entry.Payload), &payload))
	require.Equal(t, "******7890", payload["account_number"])
	require.NotContains(t, entry.Payload, "1234567890")
	require.NotContains(t, entry.Payload, "Budi")
}

func TestBankAccountValidationService_nilAuditRepoIsSafe(t *testing.T) {
	repo := &fakeBankAccountRepository{
		pending: []model.BankAccount{pendingAccount(1, "11111111")},
	}
	validator := &fakeBankAccountValidator{
		resp: &pg_provider.BankAccountValidationResponse{ID: "vid-1", Status: "verified"},
	}
	svc := NewBankAccountValidationService(repo, validator, nil)

	require.NotPanics(t, func() {
		_ = svc.ProcessPendingValidations(context.Background())
	})
}

func TestMaskAccountNumber(t *testing.T) {
	require.Equal(t, "", maskAccountNumber(""))
	require.Equal(t, "***", maskAccountNumber("123"))
	require.Equal(t, "****", maskAccountNumber("1234"))
	require.Equal(t, "****1111", maskAccountNumber("11111111"))
	require.Equal(t, "******7890", maskAccountNumber("1234567890"))
}
