package rpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	require.Equal(t, 30*time.Second, cfg.Timeout)
	require.Equal(t, 3, cfg.MaxRetries)
	require.Equal(t, uint32(5), cfg.CircuitBreaker.MaxRequests)
	require.Equal(t, 60*time.Second, cfg.CircuitBreaker.Interval)
	require.Equal(t, 30*time.Second, cfg.CircuitBreaker.Timeout)
	require.Equal(t, uint32(5), cfg.CircuitBreaker.FailureThreshold)
}

func TestClientConfigStruct(t *testing.T) {
	cfg := ClientConfig{
		URL:        "https://rpc.example.com",
		Timeout:    15 * time.Second,
		MaxRetries: 5,
		CircuitBreaker: CircuitBreakerConfig{
			MaxRequests:      10,
			Interval:         120 * time.Second,
			Timeout:          60 * time.Second,
			FailureThreshold: 3,
		},
	}

	require.Equal(t, "https://rpc.example.com", cfg.URL)
	require.Equal(t, 15*time.Second, cfg.Timeout)
	require.Equal(t, 5, cfg.MaxRetries)
	require.Equal(t, uint32(10), cfg.CircuitBreaker.MaxRequests)
	require.Equal(t, 120*time.Second, cfg.CircuitBreaker.Interval)
	require.Equal(t, 60*time.Second, cfg.CircuitBreaker.Timeout)
	require.Equal(t, uint32(3), cfg.CircuitBreaker.FailureThreshold)
}

func TestCircuitBreakerConfigStruct(t *testing.T) {
	cfg := CircuitBreakerConfig{
		MaxRequests:      7,
		Interval:         45 * time.Second,
		Timeout:          20 * time.Second,
		FailureThreshold: 10,
	}

	require.Equal(t, uint32(7), cfg.MaxRequests)
	require.Equal(t, 45*time.Second, cfg.Interval)
	require.Equal(t, 20*time.Second, cfg.Timeout)
	require.Equal(t, uint32(10), cfg.FailureThreshold)
}

func TestIsRangeTooLargeError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "query returned more than",
			err:  errors.New("query returned more than 10000 results"),
			want: true,
		},
		{
			name: "block range too large",
			err:  errors.New("block range too large"),
			want: true,
		},
		{
			name: "exceed maximum block range",
			err:  errors.New("exceed maximum block range: 10000"),
			want: true,
		},
		{
			name: "too many results",
			err:  errors.New("Error: too many results"),
			want: true,
		},
		{
			name: "range too wide",
			err:  errors.New("Error: range too wide for query"),
			want: true,
		},
		{
			name: "block range is too wide",
			err:  errors.New("block range is too wide"),
			want: true,
		},
		{
			name: "query timeout",
			err:  errors.New("query timeout exceeded"),
			want: true,
		},
		{
			name: "response too large",
			err:  errors.New("response too large"),
			want: true,
		},
		{
			name: "max results",
			err:  errors.New("max results limit reached"),
			want: true,
		},
		{
			name: "limit exceeded",
			err:  errors.New("rate limit exceeded"),
			want: true,
		},
		{
			name: "case insensitive - BLOCK RANGE TOO LARGE",
			err:  errors.New("BLOCK RANGE TOO LARGE"),
			want: true,
		},
		{
			name: "unrelated error - connection refused",
			err:  errors.New("connection refused"),
			want: false,
		},
		{
			name: "unrelated error - block not found",
			err:  errors.New("block not found"),
			want: false,
		},
		{
			name: "unrelated error - invalid params",
			err:  errors.New("invalid params"),
			want: false,
		},
		{
			name: "unrelated error - context canceled",
			err:  errors.New("context canceled"),
			want: false,
		},
		{
			name: "mixed case error",
			err:  errors.New("Error: Query Returned More Than 5000 results"),
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isRangeTooLargeError(tc.err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestDefaultConfigValues(t *testing.T) {
	cfg := DefaultConfig()

	// Verify URL is empty by default (must be set by user)
	require.Empty(t, cfg.URL)

	// Verify circuit breaker has sensible defaults
	require.Greater(t, cfg.CircuitBreaker.MaxRequests, uint32(0))
	require.Greater(t, cfg.CircuitBreaker.Interval, time.Duration(0))
	require.Greater(t, cfg.CircuitBreaker.Timeout, time.Duration(0))
	require.Greater(t, cfg.CircuitBreaker.FailureThreshold, uint32(0))

	// Verify timeout and retries are set
	require.Greater(t, cfg.Timeout, time.Duration(0))
	require.Greater(t, cfg.MaxRetries, 0)
}

func TestClientConfigZeroValues(t *testing.T) {
	var cfg ClientConfig

	// Zero values should be empty/zero
	require.Empty(t, cfg.URL)
	require.Zero(t, cfg.Timeout)
	require.Zero(t, cfg.MaxRetries)
	require.Zero(t, cfg.CircuitBreaker.MaxRequests)
	require.Zero(t, cfg.CircuitBreaker.Interval)
	require.Zero(t, cfg.CircuitBreaker.Timeout)
	require.Zero(t, cfg.CircuitBreaker.FailureThreshold)
}

func TestNewClientWithInvalidURL(t *testing.T) {
	cfg := DefaultConfig()
	cfg.URL = "not-a-valid-url"

	_, err := New(context.Background(), cfg)
	require.Error(t, err)
}

func TestNewClientWithEmptyURL(t *testing.T) {
	cfg := DefaultConfig()
	cfg.URL = ""

	_, err := New(context.Background(), cfg)
	require.Error(t, err)
}

func TestIsRangeTooLargeErrorEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "empty error message",
			err:  errors.New(""),
			want: false,
		},
		{
			name: "partial match - just 'range'",
			err:  errors.New("range error"),
			want: false,
		},
		{
			name: "partial match - just 'too'",
			err:  errors.New("too bad"),
			want: false,
		},
		{
			name: "partial match - just 'large'",
			err:  errors.New("large response"),
			want: false,
		},
		{
			name: "alchemy style error",
			err:  errors.New("Log response size exceeded. You can make eth_getLogs requests with up to a 2K block range"),
			want: false, // Not matching current patterns
		},
		{
			name: "infura style - query returned more than 10000 results",
			err:  errors.New("query returned more than 10000 results; try with this block range"),
			want: true,
		},
		{
			name: "quicknode style - exceed maximum block range",
			err:  errors.New("exceed maximum block range: 10000"),
			want: true,
		},
		{
			name: "wrapped error containing indicator",
			err:  errors.New("eth_getLogs failed: block range too large"),
			want: true,
		},
		{
			name: "json rpc error format",
			err:  errors.New(`{"code":-32005,"message":"query returned more than 10000 results"}`),
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isRangeTooLargeError(tc.err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestCircuitBreakerConfigValidDefaults(t *testing.T) {
	cfg := DefaultConfig()

	// Circuit breaker should have production-ready defaults
	// MaxRequests: number of requests allowed in half-open state
	require.Equal(t, uint32(5), cfg.CircuitBreaker.MaxRequests)

	// Interval: cyclic period of the closed state (resets counters)
	require.Equal(t, 60*time.Second, cfg.CircuitBreaker.Interval)

	// Timeout: period of the open state before going half-open
	require.Equal(t, 30*time.Second, cfg.CircuitBreaker.Timeout)

	// FailureThreshold: consecutive failures before opening
	require.Equal(t, uint32(5), cfg.CircuitBreaker.FailureThreshold)
}

func TestClientConfigWithCustomValues(t *testing.T) {
	cfg := ClientConfig{
		URL:        "https://linea-mainnet.infura.io/v3/API_KEY",
		Timeout:    45 * time.Second,
		MaxRetries: 10,
		CircuitBreaker: CircuitBreakerConfig{
			MaxRequests:      20,
			Interval:         30 * time.Second,
			Timeout:          15 * time.Second,
			FailureThreshold: 3,
		},
	}

	require.Contains(t, cfg.URL, "infura.io")
	require.Equal(t, 45*time.Second, cfg.Timeout)
	require.Equal(t, 10, cfg.MaxRetries)
	require.Equal(t, uint32(20), cfg.CircuitBreaker.MaxRequests)
}
