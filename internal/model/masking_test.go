package model

import "testing"

func TestMaskAccountNumber(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty", input: "", expected: ""},
		{name: "shorter than four", input: "123", expected: "***"},
		{name: "exactly four", input: "1234", expected: "****"},
		{name: "eight digits", input: "11111111", expected: "****1111"},
		{name: "ten digits", input: "1234567890", expected: "******7890"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MaskAccountNumber(tc.input); got != tc.expected {
				t.Fatalf("MaskAccountNumber(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}
