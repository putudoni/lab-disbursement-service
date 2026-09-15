package model

import (
	"net/http"
	"time"
)

type DisbursementNotFoundError struct{}

func (DisbursementNotFoundError) Error() string { return "disbursement not found" }

func (DisbursementNotFoundError) StatusCode() int { return http.StatusNotFound }

var ErrDisbursementNotFound DisbursementNotFoundError

type InvalidDisbursementIDError struct{}

func (InvalidDisbursementIDError) Error() string { return "invalid disbursement id" }

func (InvalidDisbursementIDError) StatusCode() int { return http.StatusBadRequest }

var ErrInvalidDisbursementID InvalidDisbursementIDError

type Disbursement struct {
	ID                 int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	LoanID             int64      `gorm:"index" json:"loan_id"`
	BankAccountID      int64      `gorm:"index" json:"bank_account_id"`
	BankCode           string     `gorm:"size:10;not null" json:"bank_code"`
	AccountNumber      string     `gorm:"size:50;not null" json:"account_number"`
	AccountHolderName  string     `gorm:"size:255;not null" json:"account_holder_name"`
	Description        string     `gorm:"size:255;not null" json:"description"`
	Amount             int64      `gorm:"not null" json:"amount"`
	DisbursementStatus string     `gorm:"size:20;not null;default:pending" json:"disbursement_status"`
	ExternalID         string     `gorm:"size:100" json:"external_id"`
	ProviderID         string     `gorm:"size:100" json:"provider_id"`
	ProcessedAt        *time.Time `json:"processed_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

const (
	DisbursementStatusPending    = "pending"
	DisbursementStatusProcessing = "processing"
	DisbursementStatusSuccess    = "success"
	DisbursementStatusFailed     = "failed"
	DisbursementStatusError      = "error"
)
