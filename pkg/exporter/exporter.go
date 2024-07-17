package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

	e.git, err = git.New(e.cfg.RepoClonePath, e.cfg.SSHURL, e.cfg.SSHKey, e.cfg.SshKeyPassword, e.cfg.SshKnownHostsPath, e.cfg.SshAcceptUnknownHosts)
	if err != nil {
		return fmt.Errorf("failed to create Git client: %w", err)
	}

	e.grafana, err = grafana.New(e.cfg.GrafanaURL, e.cfg.GrafanaApiKey)
	if err != nil {
		return fmt.Errorf("failed to create Grafana client: %w", err)
	}

	branchName, err := e.git.CheckoutNewBranch(e.cfg.BaseBranch, e.cfg.BranchPrefix)
	if err != nil {
		return fmt.Errorf("failed to checkout new branch: %w", err)
	}

	dashboards, err := e.grafana.ListAndExportDashboards(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list and export dashboards: %w", err)
	}

	diffedDashboards, err := e.getDiffedDashboards(dashboards)
	if err != nil {
		return fmt.Errorf("failed to get diffed dashboards: %w", err)
	}

	if diffedDashboards == nil {
		return nil
	}

	if err := e.saveDashboards(diffedDashboards); err != nil {
		return fmt.Errorf("failed to save dashboards: %w", err)
	}

	if err := e.git.CommitAll(e.cfg.SSHUser, e.cfg.SSHUser); err != nil {
		return fmt.Errorf("failed to commit and push changes: %w", err)
	}

	if err := e.git.Push(branchName); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	e.logger.Info().Str("branch", branchName).Msg("Pushed changes to branch")
	return nil
}

func (e *Exporter) getDiffedDashboards(dashboards []sdk.Board) ([]sdk.Board, error) {
	var diffedDashboards []sdk.Board
	for _, dashboard := range dashboards {
		dashboardJson, err := json.MarshalIndent(dashboard, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal dashboard: %w", err)
		}

		repoDashboardJson, err := os.ReadFile(filepath.Join(e.cfg.RepoSavePath, fmt.Sprintf("%s.json", dashboard.UID)))
		if err != nil {
			diffedDashboards = append(diffedDashboards, dashboard)
			continue
		}
		if string(dashboardJson) != string(repoDashboardJson) {
			diffedDashboards = append(diffedDashboards, dashboard)
		}
	}

	if len(diffedDashboards) == 0 {
		e.logger.Info().Msg("No changes detected in dashboards")
		return nil, nil
	} else {
		e.logger.Info().Msg("Changes detected in dashboards")
	}

	return diffedDashboards, nil
}

func (e *Exporter) saveDashboards(dashboards []sdk.Board) error {
	if err := os.MkdirAll(filepath.Dir(e.cfg.RepoSavePath), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	for _, dashboard := range dashboards {
		dashboardJson, err := json.MarshalIndent(dashboard, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal dashboard: %w", err)
		}

		if err := os.WriteFile(filepath.Join(e.cfg.RepoSavePath, fmt.Sprintf("%s.json", dashboard.UID)), dashboardJson, 0644); err != nil {
			return fmt.Errorf("failed to write dashboard file: %w", err)
		}

		e.logger.Info().Str("dashboard", dashboard.UID).Msg("Saved dashboard")
	}
	return nil
}
