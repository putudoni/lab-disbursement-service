package pg_provider

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsTransient(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "non-provider error", err: errors.New("connection refused"), want: true},
		{name: "wrapped non-provider error", err: errors.Join(errors.New("boom"), errors.New("connection reset")), want: true},
		{name: "provider 5xx", err: &ProviderError{StatusCode: 502, ErrorCode: "PROVIDER_UNAVAILABLE"}, want: true},
		{name: "provider 4xx", err: &ProviderError{StatusCode: 404, ErrorCode: "BANK_ACCOUNT_NOT_FOUND"}, want: false},
		{name: "provider no status", err: &ProviderError{ErrorCode: "LOCAL_ERROR"}, want: false},
		{name: "provider wrapped", err: errors.Join(&ProviderError{StatusCode: 500}), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, IsTransient(tt.err))
		})
	}
}

func TestIsPermanent(t *testing.T) {
	require.False(t, IsPermanent(nil))
	require.False(t, IsPermanent(errors.New("timeout")))
	require.True(t, IsPermanent(&ProviderError{StatusCode: 400}))
	require.True(t, IsPermanent(NewLocalError("REQUEST_ERROR", "marshal failed")))
}
