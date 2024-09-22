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

func New(cfg *config.Config, logger zerolog.Logger) (*Exporter, error) {
	gitClient, err := git.New(cfg.RepoClonePath, cfg.SSHURL, cfg.SSHKey, cfg.SshKeyPassword, cfg.SshKnownHostsPath, cfg.SshAcceptUnknownHosts)
	if err != nil {
		return nil, fmt.Errorf("failed to create Git client: %w", err)
	}

	grafanaClient, err := grafana.New(cfg.GrafanaURL, cfg.GrafanaSaToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Grafana client: %w", err)
	}

	return &Exporter{
		cfg:     cfg,
		git:     gitClient,
		grafana: grafanaClient,
		logger:  logger,
	}, nil
}

func (e *Exporter) Run(ctx context.Context) error {
	branchName, err := e.git.CheckoutNewBranch(ctx, e.cfg.BaseBranch, e.cfg.BranchPrefix)
	if err != nil {
		return fmt.Errorf("failed to checkout new branch: %w", err)
	}

	dashboards, err := e.grafana.ListAndExportDashboards(ctx)
	if err != nil {
		return fmt.Errorf("failed to list and export dashboards: %w", err)
	}

	diffedDashboards, err := e.getDiffedDashboards(ctx, dashboards)
	if err != nil {
		return fmt.Errorf("failed to get diffed dashboards: %w", err)
	}

	if len(diffedDashboards) == 0 {
		e.logger.Info().Msg("no changes detected in dashboards")
		return nil
	}

	if err := e.saveDashboards(ctx, diffedDashboards); err != nil {
		return fmt.Errorf("failed to save dashboards: %w", err)
	}

	if err := e.git.CommitAll(ctx, e.cfg.SSHUser, e.cfg.SSHEmail); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	if err := e.git.Push(ctx, branchName); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	e.logger.Info().Str("branch", branchName).Msg("pushed changes to branch")
	return nil
}

func (e *Exporter) getDiffedDashboards(ctx context.Context, dashboards []sdk.Board) ([]sdk.Board, error) {
	var diffedDashboards []sdk.Board
	for _, dashboard := range dashboards {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			dashboardJson, err := json.MarshalIndent(dashboard, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal dashboard: %w", err)
			}

			repoDashboardPath := filepath.Join(e.cfg.RepoSavePath, fmt.Sprintf("%s.json", dashboard.UID))
			repoDashboardJson, err := os.ReadFile(repoDashboardPath)
			if err != nil {
				if !os.IsNotExist(err) {
					return nil, fmt.Errorf("failed to read existing dashboard file: %w", err)
				}
				diffedDashboards = append(diffedDashboards, dashboard)
				continue
			}

			if string(dashboardJson) != string(repoDashboardJson) {
				diffedDashboards = append(diffedDashboards, dashboard)
			}
		}
	}

	return diffedDashboards, nil
}

func (e *Exporter) saveDashboards(ctx context.Context, dashboards []sdk.Board) error {
	fullSavePath := e.cfg.RepoSavePath

	if err := os.MkdirAll(fullSavePath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", fullSavePath, err)
	}

	for _, dashboard := range dashboards {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			dashboardJson, err := json.MarshalIndent(dashboard, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal dashboard: %w", err)
			}

			filePath := filepath.Join(fullSavePath, fmt.Sprintf("%s.json", dashboard.UID))

			e.logger.Info().Str("path", filePath).Msg("Writing dashboard file")

			if err := os.WriteFile(filePath, dashboardJson, 0644); err != nil {
				return fmt.Errorf("failed to write dashboard file %s: %w", filePath, err)
			}

			e.logger.Info().
				Str("dashboard", dashboard.UID).
				Str("path", filePath).
				Msg("saved dashboard")
		}
	}
	return nil
}
