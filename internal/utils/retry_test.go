package utils

import (
	"context"
	"errors"
	"testing"
	"time"

	"grafana-db-exporter/internal/config"
	"grafana-db-exporter/internal/logger"
)

func TestRetry(t *testing.T) {
	err := logger.Init()
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	tests := []struct {
		name           string
		cfg            *config.Config
		operation      string
		fn             func() (interface{}, error)
		expectedResult interface{}
		expectedError  bool
		expectedCalls  int
	}{
		{
			name: "Successful on first try",
			cfg: &config.Config{
				EnableRetries:  true,
				NumOfRetries:   3,
				RetriesBackoff: 1,
			},
			operation: "test operation",
			fn: func() (interface{}, error) {
				return "success", nil
			},
			expectedResult: "success",
			expectedError:  false,
			expectedCalls:  1,
		},
		{
			name: "Successful after retries",
			cfg: &config.Config{
				EnableRetries:  true,
				NumOfRetries:   3,
				RetriesBackoff: 1,
			},
			operation: "test operation",
			fn: func() func() (interface{}, error) {
				count := 0
				return func() (interface{}, error) {
					count++
					if count < 3 {
						return nil, errors.New("temporary error")
					}
					return "success", nil
				}
			}(),
			expectedResult: "success",
			expectedError:  false,
			expectedCalls:  3,
		},
		{
			name: "Failure after all retries",
			cfg: &config.Config{
				EnableRetries:  true,
				NumOfRetries:   3,
				RetriesBackoff: 1,
			},
			operation: "test operation",
			fn: func() (interface{}, error) {
				return nil, errors.New("persistent error")
			},
			expectedResult: nil,
			expectedError:  true,
			expectedCalls:  3,
		},
		{
			name: "Retries disabled",
			cfg: &config.Config{
				EnableRetries:  false,
				NumOfRetries:   3,
				RetriesBackoff: 1,
			},
			operation: "test operation",
			fn: func() (interface{}, error) {
				return nil, errors.New("error")
			},
			expectedResult: nil,
			expectedError:  true,
			expectedCalls:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			calls := 0
			wrappedFn := func() (interface{}, error) {
				calls++
				return tt.fn()
			}

			result, err := Retry(ctx, tt.cfg, tt.operation, wrappedFn)

			if (err != nil) != tt.expectedError {
				t.Errorf("Retry() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			if result != tt.expectedResult {
				t.Errorf("Retry() result = %v, expectedResult %v", result, tt.expectedResult)
			}

			if calls != tt.expectedCalls {
				t.Errorf("Retry() calls = %d, expectedCalls %d", calls, tt.expectedCalls)
			}
		})
	}
}

func TestRetryWithContext(t *testing.T) {
	err := logger.Init()
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	cfg := &config.Config{
		EnableRetries:  true,
		NumOfRetries:   5,
		RetriesBackoff: 1,
	}

	t.Run("Context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		calls := 0
		fn := func() (interface{}, error) {
			calls++
			if calls == 2 {
				cancel()
			}
			return nil, errors.New("test error")
		}

		_, err := Retry(ctx, cfg, "test operation", fn)

		if err != context.Canceled {
			t.Errorf("Expected context.Canceled error, got %v", err)
		}

		if calls != 2 {
			t.Errorf("Expected 2 calls, got %d", calls)
		}
	})
}

func TestOperationError(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		err       error
		expected  string
	}{
		{
			name:      "Basic error",
			operation: "test operation",
			err:       errors.New("test error"),
			expected:  "test operation failed: test error",
		},
		{
			name:      "Empty operation",
			operation: "",
			err:       errors.New("error without operation"),
			expected:  " failed: error without operation",
		},
		{
			name:      "Nil error",
			operation: "nil error operation",
			err:       nil,
			expected:  "nil error operation failed: <nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &OperationError{
				Operation: tt.operation,
				Err:       tt.err,
			}

			if err.Error() != tt.expected {
				t.Errorf("OperationError.Error() = %v, want %v", err.Error(), tt.expected)
			}
		})
	}
}
