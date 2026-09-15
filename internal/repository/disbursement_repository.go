package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"lab-disbursement-service/internal/model"
)

type DisbursementRepository struct {
	db GormDBProvider
}

func NewDisbursementRepository(db GormDBProvider) *DisbursementRepository {
	return &DisbursementRepository{db: db}
}

func (r *DisbursementRepository) FindPendingOrProcessing() ([]model.Disbursement, error) {
	var disbursements []model.Disbursement

	err := r.db.DB().Where(
		"disbursement_status IN ?",
		[]string{model.DisbursementStatusPending, model.DisbursementStatusProcessing},
	).Order("id ASC").Find(&disbursements).Error
	if err != nil {
		return nil, err
	}

	return disbursements, nil
}

func (r *DisbursementRepository) FindBankAccountByID(id int64) (*model.BankAccount, error) {
	var account model.BankAccount

	err := r.db.DB().First(&account, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &account, nil
}

func (r *DisbursementRepository) UpdateDisbursementStatus(id int64, status string, providerID string) error {
	updates := map[string]interface{}{
		"disbursement_status": status,
		"provider_id":         providerID,
	}

	if status == model.DisbursementStatusSuccess {
		now := time.Now()
		updates["processed_at"] = &now
	}

	return r.db.DB().Model(&model.Disbursement{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *DisbursementRepository) FindByID(id int64) (*model.Disbursement, error) {
	var disbursement model.Disbursement

	err := r.db.DB().First(&disbursement, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &disbursement, nil
}
