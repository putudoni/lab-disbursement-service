package model

import (
	"net/http"
	"time"
)

type BankAccountNotFoundError struct{}

func (BankAccountNotFoundError) Error() string { return "bank account not found" }

func (BankAccountNotFoundError) StatusCode() int { return http.StatusNotFound }

var ErrBankAccountNotFound BankAccountNotFoundError

type InvalidBankAccountIDError struct{}

func (InvalidBankAccountIDError) Error() string { return "invalid bank account id" }

func (InvalidBankAccountIDError) StatusCode() int { return http.StatusBadRequest }

var ErrInvalidBankAccountID InvalidBankAccountIDError

type BankAccount struct {
	ID                int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	LoanID            int64      `gorm:"index" json:"loan_id"`
	BankCode          string     `gorm:"size:10;not null" json:"bank_code"`
	AccountNumber     string     `gorm:"size:50;not null" json:"account_number"`
	AccountHolderName string     `gorm:"size:255;not null" json:"account_holder_name"`
	ValidationStatus  string     `gorm:"size:20;not null;default:pending" json:"validation_status"`
	ValidationID      string     `gorm:"size:100" json:"validation_id"`
	ValidatedAt       *time.Time `json:"validated_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

const (
	ValidationStatusPending   = "pending"
	ValidationStatusValidated = "validated"
	ValidationStatusInvalid   = "invalid"
	ValidationStatusError     = "error"
)
