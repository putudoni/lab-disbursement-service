package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"lab-disbursement-service/internal/model"
	"lab-disbursement-service/pkg/pg_provider"
)

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
	svc := NewBankAccountValidationService(repo, validator)

	err := svc.ProcessPendingValidations(context.Background())

	require.NoError(t, err)
	require.Equal(t, 1, validator.calls)
	require.Equal(t, "11111111", validator.last.BankAccountNumber)
	require.Equal(t, "BCA", validator.last.BankCode)
	require.Equal(t, "Budi", validator.last.AccountHolderName)
	require.Equal(t, []validationStatusUpdate{{id: 1, status: model.ValidationStatusValidated, validationID: "vid-1"}}, repo.updates)
}

func TestProcessPendingValidations_providerErrorMarksInvalid(t *testing.T) {
	repo := &fakeBankAccountRepository{
		pending: []model.BankAccount{pendingAccount(2, "22222222")},
	}
	validator := &fakeBankAccountValidator{err: errors.New("provider unavailable")}
	svc := NewBankAccountValidationService(repo, validator)

	err := svc.ProcessPendingValidations(context.Background())

	require.NoError(t, err)
	require.Equal(t, []validationStatusUpdate{{id: 2, status: model.ValidationStatusInvalid, validationID: ""}}, repo.updates)
}

func TestProcessPendingValidations_unverifiedAccountMarksInvalid(t *testing.T) {
	repo := &fakeBankAccountRepository{
		pending: []model.BankAccount{pendingAccount(3, "99999999")},
	}
	validator := &fakeBankAccountValidator{
		resp: &pg_provider.BankAccountValidationResponse{ID: "vid-3", Status: "unverified"},
	}
	svc := NewBankAccountValidationService(repo, validator)

	err := svc.ProcessPendingValidations(context.Background())

	require.NoError(t, err)
	require.Equal(t, []validationStatusUpdate{{id: 3, status: model.ValidationStatusInvalid, validationID: "vid-3"}}, repo.updates)
}

func TestProcessPendingValidations_skipsAccountsThatAlreadyHaveFinalStatus(t *testing.T) {
	validated := pendingAccount(1, "11111111")
	validated.ValidationStatus = model.ValidationStatusValidated
	invalid := pendingAccount(2, "22222222")
	invalid.ValidationStatus = model.ValidationStatusInvalid

	repo := &fakeBankAccountRepository{
		pending: []model.BankAccount{validated, invalid},
	}
	validator := &fakeBankAccountValidator{
		resp: &pg_provider.BankAccountValidationResponse{ID: "vid-1", Status: "verified"},
	}
	svc := NewBankAccountValidationService(repo, validator)

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
	svc := NewBankAccountValidationService(repo, validator)

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
	svc := NewBankAccountValidationService(repo, validator)

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
	svc := NewBankAccountValidationService(repo, &fakeBankAccountValidator{})

	account, err := svc.GetBankAccount(context.Background(), 1)

	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(1), account.ID)
}

func TestGetBankAccount_notFound(t *testing.T) {
	repo := &fakeBankAccountRepository{byID: map[int64]*model.BankAccount{}}
	svc := NewBankAccountValidationService(repo, &fakeBankAccountValidator{})

	account, err := svc.GetBankAccount(context.Background(), 1)

	require.NoError(t, err)
	require.Nil(t, account)
}

func TestGetBankAccount_repoError(t *testing.T) {
	repo := &fakeBankAccountRepository{byIDErr: errors.New("db down")}
	svc := NewBankAccountValidationService(repo, &fakeBankAccountValidator{})

	account, err := svc.GetBankAccount(context.Background(), 1)

	require.Error(t, err)
	require.Nil(t, account)
}