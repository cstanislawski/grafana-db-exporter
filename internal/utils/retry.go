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
		result, err = fn()
		if err != nil {
			return result, &OperationError{Operation: operation, Err: err}
		}
		return result, nil
	}

	for i := uint(0); i < cfg.NumOfRetries; i++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
			result, err = fn()
			if err == nil {
				return result, nil
			}

			logger.Log.Error().Err(err).Uint("attempt", i+1).Uint("max_attempts", cfg.NumOfRetries).Msgf("%s failed, retrying...", operation)

			if i < cfg.NumOfRetries-1 {
				backoff := time.Duration(cfg.RetriesBackoff*(i+1)) * time.Second
				logger.Log.Info().Dur("backoff", backoff).Msgf("Waiting before next retry")
				time.Sleep(backoff)
			}
		}
	}

	logger.Log.Error().Err(err).Msgf("%s failed after all retries", operation)
	return result, &OperationError{Operation: operation, Err: err}
}
