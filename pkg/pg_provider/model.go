package pg_provider

type BankAccountValidationRequest struct {
	BankAccountNumber string `json:"bank_account_number"`
	BankCode          string `json:"bank_code"`
	AccountHolderName string `json:"name"`
}

type BankAccountValidationResponse struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	NameMatchResult   string `json:"name_match_result"`
	AccountNumber     string `json:"account_number"`
	BankCode          string `json:"bank_code"`
	AccountHolderName string `json:"name"`
}

type DisbursementRequest struct {
	ExternalID        string `json:"external_id"`
	BankCode          string `json:"bank_code"`
	AccountHolderName string `json:"account_holder_name"`
	AccountNumber     string `json:"account_number"`
	Description       string `json:"description"`
	Amount            int64  `json:"amount"`
}

type DisbursementResponse struct {
	ID          string `json:"id"`
	ExternalID  string `json:"external_id"`
	Status      string `json:"status"`
	Amount      int64  `json:"amount"`
	FailureCode string `json:"failure_code"`
}

const (
	DisbursementStatusCompleted = "COMPLETED"
	DisbursementStatusFailed    = "FAILED"
	DisbursementStatusPending   = "PENDING"
)

type ProviderError struct {
	StatusCode int    `json:"status_code"`
	ErrorCode  string `json:"error_code"`
	Message    string `json:"message"`
}

func (e *ProviderError) Error() string {
	if e.Message == "" {
		return e.ErrorCode
	}
	return e.ErrorCode + ": " + e.Message
}
