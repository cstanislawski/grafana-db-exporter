package main

import (
	"os"

	"grafana-db-exporter/pkg/config"
	"grafana-db-exporter/pkg/exporter"

	"github.com/rs/zerolog"
)

func main() {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	exp := exporter.New(cfg, logger)
	if err := exp.Run(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run exporter")
	}

	logger.Info().Msg("Successfully exported and pushed Grafana dashboards")
}
