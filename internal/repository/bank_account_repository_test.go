package repository

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"lab-disbursement-service/internal/model"
)

type testDBProvider struct {
	db *gorm.DB
}

func (p testDBProvider) DB() *gorm.DB { return p.db }

func newMockRepo(t *testing.T) (*BankAccountRepository, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	require.NoError(t, err)

	return NewBankAccountRepository(testDBProvider{db: gormDB}), mock
}

var bankAccountColumns = []string{
	"id",
	"loan_id",
	"bank_code",
	"account_number",
	"account_holder_name",
	"validation_status",
	"validation_id",
	"validated_at",
	"created_at",
	"updated_at",
}

func bankAccountRow(id int64, accountNumber string, status string) []driver.Value {
	created := time.Date(2026, 9, 9, 15, 0, 0, 0, time.UTC)

	var (
		validationID any = nil
		validatedAt  any = nil
	)
	if status == model.ValidationStatusValidated {
		validationID = fmt.Sprintf("vid-%d", id)
		validatedAt = time.Date(2026, 9, 9, 15, 5, 0, 0, time.UTC)
	}

	return []driver.Value{id, id * 10, "BCA", accountNumber, "Budi", status, validationID, validatedAt, created, created}
}

func TestFindPendingValidation(t *testing.T) {
	repo, mock := newMockRepo(t)

	const query = `SELECT * FROM "bank_accounts" WHERE validation_status = $1 ORDER BY id ASC`

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(model.ValidationStatusPending).
		WillReturnRows(sqlmock.NewRows(bankAccountColumns).
			AddRow(bankAccountRow(1, "11111111", model.ValidationStatusPending)...).
			AddRow(bankAccountRow(2, "22222222", model.ValidationStatusValidated)...))

	accounts, err := repo.FindPendingValidation()

	require.NoError(t, err)
	require.Len(t, accounts, 2)
	require.Equal(t, int64(1), accounts[0].ID)
	require.Equal(t, "11111111", accounts[0].AccountNumber)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindPendingValidation_errorIsReturned(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(`SELECT \* FROM "bank_accounts".*`).
		WillReturnError(errors.New("db down"))

	accounts, err := repo.FindPendingValidation()

	require.Error(t, err)
	require.Nil(t, accounts)
	require.NoError(t, mock.ExpectationsWereMet())
}

const updateQueryPattern = `^UPDATE "bank_accounts" SET .* WHERE id = \$[0-9]+$`

func TestUpdateValidationStatus_validated(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectBegin()
	mock.ExpectExec(updateQueryPattern).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.UpdateValidationStatus(1, model.ValidationStatusValidated, "vid-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateValidationStatus_invalid(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectBegin()
	mock.ExpectExec(updateQueryPattern).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.UpdateValidationStatus(1, model.ValidationStatusInvalid, "vid-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByID_found(t *testing.T) {
	repo, mock := newMockRepo(t)

	const query = `SELECT * FROM "bank_accounts" WHERE "bank_accounts"."id" = $1 ORDER BY "bank_accounts"."id" LIMIT $2`

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(int64(1), int64(1)).
		WillReturnRows(sqlmock.NewRows(bankAccountColumns).
			AddRow(bankAccountRow(1, "11111111", model.ValidationStatusValidated)...))

	account, err := repo.FindByID(1)

	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(1), account.ID)
	require.Equal(t, model.ValidationStatusValidated, account.ValidationStatus)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByID_notFound(t *testing.T) {
	repo, mock := newMockRepo(t)

	const query = `SELECT * FROM "bank_accounts" WHERE "bank_accounts"."id" = $1 ORDER BY "bank_accounts"."id" LIMIT $2`

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(int64(1), int64(1)).
		WillReturnRows(sqlmock.NewRows(bankAccountColumns))

	account, err := repo.FindByID(1)

	require.NoError(t, err)
	require.Nil(t, account)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByID_errorIsReturned(t *testing.T) {
	repo, mock := newMockRepo(t)

	const query = `SELECT * FROM "bank_accounts" WHERE "bank_accounts"."id" = $1 ORDER BY "bank_accounts"."id" LIMIT $2`

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(int64(1), int64(1)).
		WillReturnError(errors.New("db down"))

	account, err := repo.FindByID(1)

	require.Error(t, err)
	require.Nil(t, account)
	require.NoError(t, mock.ExpectationsWereMet())
}
