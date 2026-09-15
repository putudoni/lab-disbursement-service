package repository

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"lab-disbursement-service/internal/model"
)

func newMockDisbursementRepo(t *testing.T) (*DisbursementRepository, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	require.NoError(t, err)

	return NewDisbursementRepository(testDBProvider{db: gormDB}), mock
}

var disbursementColumns = []string{
	"id",
	"loan_id",
	"bank_account_id",
	"bank_code",
	"account_number",
	"account_holder_name",
	"amount",
	"disbursement_status",
	"external_id",
	"provider_id",
	"processed_at",
	"created_at",
	"updated_at",
}

func TestFindPendingOrProcessing(t *testing.T) {
	repo, mock := newMockDisbursementRepo(t)

	mock.ExpectQuery(`^SELECT \* FROM "disbursements" WHERE disbursement_status IN .* ORDER BY id ASC$`).
		WillReturnRows(sqlmock.NewRows(disbursementColumns).
			AddRow(1, 101, 1, "BCA", "11111111", "Budi", 500000, model.DisbursementStatusPending, "dsb-1", "", nil, time.Now(), time.Now()).
			AddRow(2, 102, 2, "BCA", "22222222", "Siti", 750000, model.DisbursementStatusProcessing, "dsb-2", "trx-2", nil, time.Now(), time.Now()))

	disbursements, err := repo.FindPendingOrProcessing()

	require.NoError(t, err)
	require.Len(t, disbursements, 2)
	require.Equal(t, int64(1), disbursements[0].ID)
	require.Equal(t, model.DisbursementStatusPending, disbursements[0].DisbursementStatus)
	require.Equal(t, model.DisbursementStatusProcessing, disbursements[1].DisbursementStatus)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindPendingOrProcessing_errorIsReturned(t *testing.T) {
	repo, mock := newMockDisbursementRepo(t)

	mock.ExpectQuery(`^SELECT \* FROM "disbursements" WHERE disbursement_status IN .* ORDER BY id ASC$`).
		WillReturnError(errors.New("db down"))

	disbursements, err := repo.FindPendingOrProcessing()

	require.Error(t, err)
	require.Nil(t, disbursements)
	require.NoError(t, mock.ExpectationsWereMet())
}

const updateDisbursementQueryPattern = `^UPDATE "disbursements" SET .* WHERE id = \$\d+$`

func TestUpdateDisbursementStatus_success(t *testing.T) {
	repo, mock := newMockDisbursementRepo(t)

	mock.ExpectBegin()
	mock.ExpectExec(updateDisbursementQueryPattern).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.UpdateDisbursementStatus(1, model.DisbursementStatusSuccess, "trx-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateDisbursementStatus_pending(t *testing.T) {
	repo, mock := newMockDisbursementRepo(t)

	mock.ExpectBegin()
	mock.ExpectExec(updateDisbursementQueryPattern).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.UpdateDisbursementStatus(1, model.DisbursementStatusProcessing, "trx-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindBankAccountByID_found(t *testing.T) {
	repo, mock := newMockDisbursementRepo(t)

	const query = `SELECT * FROM "bank_accounts" WHERE "bank_accounts"."id" = $1 ORDER BY "bank_accounts"."id" LIMIT $2`

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(int64(1), int64(1)).
		WillReturnRows(sqlmock.NewRows(bankAccountColumns).
			AddRow(bankAccountRow(1, "11111111", model.ValidationStatusValidated)...))

	account, err := repo.FindBankAccountByID(1)

	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, model.ValidationStatusValidated, account.ValidationStatus)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindBankAccountByID_notFound(t *testing.T) {
	repo, mock := newMockDisbursementRepo(t)

	const query = `SELECT * FROM "bank_accounts" WHERE "bank_accounts"."id" = $1 ORDER BY "bank_accounts"."id" LIMIT $2`

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(int64(1), int64(1)).
		WillReturnRows(sqlmock.NewRows(bankAccountColumns))

	account, err := repo.FindBankAccountByID(1)

	require.NoError(t, err)
	require.Nil(t, account)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindBankAccountByID_errorIsReturned(t *testing.T) {
	repo, mock := newMockDisbursementRepo(t)

	const query = `SELECT * FROM "bank_accounts" WHERE "bank_accounts"."id" = $1 ORDER BY "bank_accounts"."id" LIMIT $2`

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(int64(1), int64(1)).
		WillReturnError(errors.New("db down"))

	account, err := repo.FindBankAccountByID(1)

	require.Error(t, err)
	require.Nil(t, account)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindDisbursementByID_found(t *testing.T) {
	repo, mock := newMockDisbursementRepo(t)

	const query = `SELECT * FROM "disbursements" WHERE "disbursements"."id" = $1 ORDER BY "disbursements"."id" LIMIT $2`

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(int64(1), int64(1)).
		WillReturnRows(sqlmock.NewRows(disbursementColumns).
			AddRow(1, 101, 1, "BCA", "11111111", "Budi", 500000, model.DisbursementStatusSuccess, "dsb-1", "trx-1", time.Now(), time.Now(), time.Now()))

	disbursement, err := repo.FindByID(1)

	require.NoError(t, err)
	require.NotNil(t, disbursement)
	require.Equal(t, int64(1), disbursement.ID)
	require.Equal(t, model.DisbursementStatusSuccess, disbursement.DisbursementStatus)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindDisbursementByID_notFound(t *testing.T) {
	repo, mock := newMockDisbursementRepo(t)

	const query = `SELECT * FROM "disbursements" WHERE "disbursements"."id" = $1 ORDER BY "disbursements"."id" LIMIT $2`

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(int64(1), int64(1)).
		WillReturnRows(sqlmock.NewRows(disbursementColumns))

	disbursement, err := repo.FindByID(1)

	require.NoError(t, err)
	require.Nil(t, disbursement)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindDisbursementByID_errorIsReturned(t *testing.T) {
	repo, mock := newMockDisbursementRepo(t)

	const query = `SELECT * FROM "disbursements" WHERE "disbursements"."id" = $1 ORDER BY "disbursements"."id" LIMIT $2`

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(int64(1), int64(1)).
		WillReturnError(errors.New("db down"))

	disbursement, err := repo.FindByID(1)

	require.Error(t, err)
	require.Nil(t, disbursement)
	require.NoError(t, mock.ExpectationsWereMet())
}
