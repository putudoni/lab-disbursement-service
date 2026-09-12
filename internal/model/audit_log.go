package model

import "time"

const (
	AuditEntityBankAccount = "bank_account"

	AuditActionValidated       = "bank_account.validated"
	AuditActionInvalid         = "bank_account.invalid"
	AuditActionValidationError = "bank_account.validation_error"
	AuditActionViewed          = "bank_account.viewed"
)

type AuditLog struct {
	ID            int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	EntityType    string    `gorm:"size:50;not null;index" json:"entity_type"`
	EntityID      int64     `gorm:"not null;index" json:"entity_id"`
	Action        string    `gorm:"size:100;not null" json:"action"`
	Actor         string    `gorm:"size:255;not null" json:"actor"`
	CorrelationID string    `gorm:"size:100" json:"correlation_id,omitempty"`
	Payload       string    `gorm:"type:text" json:"payload"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
}
