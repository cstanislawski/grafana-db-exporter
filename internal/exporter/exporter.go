// internal/exporter/exporter.go
package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cstanislawski/grafana-db-exporter/internal/config"
	"github.com/cstanislawski/grafana-db-exporter/internal/git"
	"github.com/cstanislawski/grafana-db-exporter/internal/grafana"

	"github.com/grafana-tools/sdk"
	"github.com/rs/zerolog"
)

type Exporter struct {
	cfg    *config.Config
	git    *git.GitClient
	grf    *grafana.GrafanaClient
	logger zerolog.Logger
}

func New(cfg *config.Config, logger zerolog.Logger) *Exporter {
	return &Exporter{
		cfg:    cfg,
		logger: logger,
	}
}

func (e *Exporter) Run() error {
	var err error

	e.git, err = git.NewClient(e.cfg.SSHURL, e.cfg.SSHKey)
	if err != nil {
		return fmt.Errorf("failed to create Git client: %w", err)
	}

	e.grf, err = grafana.NewClient(e.cfg.GrafanaURL, e.cfg.GrafanaAPIKey) // Pass API key here
	if err != nil {
		return fmt.Errorf("failed to create Grafana client: %w", err)
	}

	branchName, err := e.git.CheckoutNewBranch(e.cfg.BaseBranch, e.cfg.BranchPrefix)
	if err != nil {
		return fmt.Errorf("failed to checkout new branch: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	dashboards, err := e.grf.ListAndExportDashboards(ctx)
	if err != nil {
		return fmt.Errorf("failed to list and export dashboards: %w", err)
	}

	if err := e.saveDashboards(dashboards); err != nil {
		return fmt.Errorf("failed to save dashboards: %w", err)
	}

	if err := e.git.CommitAndPush(branchName); err != nil {
		return fmt.Errorf("failed to commit and push changes: %w", err)
	}

	return nil
}

func (e *Exporter) saveDashboards(dashboards []sdk.Board) error {
	for _, dashboard := range dashboards {
		filename := filepath.Join(e.cfg.SavePath, fmt.Sprintf("%s.json", dashboard.UID))
		if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		data, err := json.MarshalIndent(dashboard, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal dashboard: %w", err)
		}

		if err := os.WriteFile(filename, data, 0644); err != nil {
			return fmt.Errorf("failed to write dashboard file: %w", err)
		}

		e.logger.Info().Str("dashboard", dashboard.UID).Msg("Saved dashboard")
	}
	return nil
}
