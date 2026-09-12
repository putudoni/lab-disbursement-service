package repository

import (
	"context"

	"lab-disbursement-service/internal/model"
)

type AuditTrailRepository struct {
	db GormDBProvider
}

func NewAuditTrailRepository(db GormDBProvider) *AuditTrailRepository {
	return &AuditTrailRepository{db: db}
}

func (r *AuditTrailRepository) Append(ctx context.Context, entry *model.AuditLog) error {
	return r.db.DB().WithContext(ctx).Create(entry).Error
}
