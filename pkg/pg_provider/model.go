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

type ProviderError struct {
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

func (e *ProviderError) Error() string {
	if e.Message == "" {
		return e.ErrorCode
	}
	return e.ErrorCode + ": " + e.Message
}
