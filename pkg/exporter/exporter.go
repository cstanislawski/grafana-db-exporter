package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"grafana-db-exporter/pkg/config"
	"grafana-db-exporter/pkg/git"
	"grafana-db-exporter/pkg/grafana"

	"github.com/grafana-tools/sdk"
	"github.com/rs/zerolog"
)

type Exporter struct {
	cfg     *config.Config
	git     *git.Client
	grafana *grafana.Client
	logger  zerolog.Logger
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

	e.grafana, err = grafana.NewClient(e.cfg.GrafanaURL, e.cfg.GrafanaAPIKey) // Pass API key here
	if err != nil {
		return fmt.Errorf("failed to create Grafana client: %w", err)
	}

	branchName, err := e.git.CheckoutNewBranch(e.cfg.BaseBranch, e.cfg.BranchPrefix)
	_ = branchName

	if err != nil {
		return fmt.Errorf("failed to checkout new branch: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	dashboards, err := e.grafana.ListAndExportDashboards(ctx)
	if err != nil {
		return fmt.Errorf("failed to list and export dashboards: %w", err)
	}

	if err := e.saveDashboards(dashboards); err != nil {
		return fmt.Errorf("failed to save dashboards: %w", err)
	}

	if err := e.git.CommitAndPush(branchName, e.cfg.SSHUser, e.cfg.SSHUser); err != nil {
		return fmt.Errorf("failed to commit and push changes: %w", err)
	}

	return nil
}

func (e *Exporter) saveDashboards(dashboards []sdk.Board) error {
	filename := filepath.Join("./repo/" + e.cfg.SavePath)
	if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	for _, dashboard := range dashboards {
		dashboardJson, err := json.MarshalIndent(dashboard, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal dashboard: %w", err)
		}

		if err := os.WriteFile(filepath.Join(filename, fmt.Sprintf("%s.json", dashboard.UID)), dashboardJson, 0644); err != nil {
			return fmt.Errorf("failed to write dashboard file: %w", err)
		}

		e.logger.Info().Str("dashboard", dashboard.UID).Msg("Saved dashboard")
	}
	return nil
}
