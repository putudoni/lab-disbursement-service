package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"lab-disbursement-service/internal/model"
)

func newMockAuditRepo(t *testing.T) (*AuditTrailRepository, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	require.NoError(t, err)

	return NewAuditTrailRepository(testDBProvider{db: gormDB}), mock
}

func TestAuditTrailRepository_Append(t *testing.T) {
	repo, mock := newMockAuditRepo(t)

	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	entry := &model.AuditLog{
		EntityType: model.AuditEntityBankAccount,
		EntityID:   1,
		Action:     model.AuditActionValidated,
		Actor:      "system",
		Payload:    `{"account_number":"****1111","to_status":"validated"}`,
		CreatedAt:  now,
	}

	const query = `INSERT INTO "audit_logs" ("entity_type","entity_id","action","actor","correlation_id","payload","created_at") VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING "id"`

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(entry.EntityType, entry.EntityID, entry.Action, entry.Actor, "", entry.Payload, entry.CreatedAt).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	err := repo.Append(context.Background(), entry)

	require.NoError(t, err)
	require.Equal(t, int64(1), entry.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}
