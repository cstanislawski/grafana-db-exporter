package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"grafana-db-exporter/pkg/config"
	"grafana-db-exporter/pkg/exporter"
	"grafana-db-exporter/pkg/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	log := logger.New()

	log.Info().Msg("Initializing grafana-db-exporter")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	exp, err := exporter.New(cfg, *log)
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	log.Info().Msg("Starting grafana-db-exporter")
	if err := exp.Run(ctx); err != nil {
		return fmt.Errorf("exporter failed: %w", err)
	}

	log.Info().Msg("Grafana-db-exporter completed successfully")
	return nil
}
