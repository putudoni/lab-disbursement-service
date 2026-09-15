package model

import "strings"

func MaskAccountNumber(accountNumber string) string {
	if len(accountNumber) <= 4 {
		return strings.Repeat("*", len(accountNumber))
	}

	return strings.Repeat("*", len(accountNumber)-4) + accountNumber[len(accountNumber)-4:]
}
