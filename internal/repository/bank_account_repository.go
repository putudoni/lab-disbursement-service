package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"lab-disbursement-service/internal/model"
)

type GormDBProvider interface {
	DB() *gorm.DB
}

type BankAccountRepository struct {
	db GormDBProvider
}

func NewBankAccountRepository(db GormDBProvider) *BankAccountRepository {
	return &BankAccountRepository{db: db}
}

func (r *BankAccountRepository) FindPendingValidation() ([]model.BankAccount, error) {
	var accounts []model.BankAccount

	err := r.db.DB().Where(
		"validation_status = ?", model.ValidationStatusPending,
	).Order("id ASC").Find(&accounts).Error
	if err != nil {
		return nil, err
	}

	return accounts, nil
}

func (r *BankAccountRepository) UpdateValidationStatus(id int64, status string, validationID string) error {
	updates := map[string]interface{}{
		"validation_status": status,
		"validation_id":     validationID,
	}

	if status == model.ValidationStatusValidated {
		now := time.Now()
		updates["validated_at"] = &now
	}

	return r.db.DB().Model(&model.BankAccount{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *BankAccountRepository) FindByID(id int64) (*model.BankAccount, error) {
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
