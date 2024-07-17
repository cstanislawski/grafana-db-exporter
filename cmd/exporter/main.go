package main

import (
	"grafana-db-exporter/pkg/config"
	"grafana-db-exporter/pkg/exporter"
	"grafana-db-exporter/pkg/logger"
)

func main() {
	logger := logger.New()

	logger.Info().Msg("Initializing grafana-db-exporter")
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	exp := exporter.New(cfg, *logger)
	if err := exp.Run(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run exporter")
	}
	logger.Info().Msg("Exiting")
}
