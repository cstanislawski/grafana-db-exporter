package utils

import (
	"context"
	"fmt"
	"time"

	"grafana-db-exporter/internal/config"
	"grafana-db-exporter/internal/logger"
)

type OperationError struct {
	Operation string
	Err       error
}

func (e *OperationError) Error() string {
	return fmt.Sprintf("%s failed: %v", e.Operation, e.Err)
}

func Retry[T any](ctx context.Context, cfg *config.Config, operation string, fn func() (T, error)) (T, error) {
	var result T
	var err error

	if !cfg.EnableRetries {
		logger.Log.Debug().Str("operation", operation).Msg("Retries disabled, executing operation once")
		result, err = fn()
		if err != nil {
			return result, &OperationError{Operation: operation, Err: err}
		}
		logger.Log.Debug().Str("operation", operation).Msg("Operation successful on first attempt")
		return result, nil
	}

	for i := uint(0); i < cfg.NumOfRetries; i++ {
		select {
		case <-ctx.Done():
			logger.Log.Debug().Str("operation", operation).Msg("Context cancelled, stopping retry attempts")
			return result, ctx.Err()
		default:
			logger.Log.Debug().Str("operation", operation).Uint("attempt", i+1).Uint("max_attempts", cfg.NumOfRetries).Msg("Attempting operation")
			result, err = fn()
			if err == nil {
				logger.Log.Debug().Str("operation", operation).Uint("attempt", i+1).Msg("Operation successful")
				return result, nil
			}

			logger.Log.Error().Err(err).Uint("attempt", i+1).Uint("max_attempts", cfg.NumOfRetries).Msgf("%s failed, retrying...", operation)

			if i < cfg.NumOfRetries-1 {
				backoff := time.Duration(cfg.RetriesBackoff*(i+1)) * time.Second
				logger.Log.Info().Str("operation", operation).Dur("backoff", backoff).Msg("Waiting before next retry")
				time.Sleep(backoff)
			}
		}
	}

	logger.Log.Error().Err(err).Msgf("%s failed after all retries", operation)
	return result, &OperationError{Operation: operation, Err: err}
}
