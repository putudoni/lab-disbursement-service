package pg_provider

import "errors"

func IsTransient(err error) bool {
	if err == nil {
		return false
	}

	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe.StatusCode >= 500
	}

	return true
}

func IsPermanent(err error) bool {
	return err != nil && !IsTransient(err)
}

func NewLocalError(errorCode, message string) *ProviderError {
	return &ProviderError{ErrorCode: errorCode, Message: message}
}
