package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"grafana-db-exporter/pkg/config"
	"grafana-db-exporter/pkg/git"
	"grafana-db-exporter/pkg/grafana"
	"grafana-db-exporter/pkg/logger"

	"github.com/grafana-tools/sdk"
	"github.com/rs/zerolog"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupSignalHandler(cancel)

	log := logger.New()

	log.Info().Msg("initializing grafana-db-exporter")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	gitClient, err := git.New(cfg.RepoClonePath, cfg.SSHURL, cfg.SSHKey, cfg.SshKeyPassword, cfg.SshKnownHostsPath, cfg.SshAcceptUnknownHosts)
	if err != nil {
		return fmt.Errorf("failed to create Git client: %w", err)
	}

	grafanaClient, err := grafana.New(cfg.GrafanaURL, cfg.GrafanaSaToken)
	if err != nil {
		return fmt.Errorf("failed to create Grafana client: %w", err)
	}

	branchName, err := gitClient.CheckoutNewBranch(ctx, cfg.BaseBranch, cfg.BranchPrefix)
	if err != nil {
		return fmt.Errorf("failed to checkout new branch: %w", err)
	}

	dashboards, err := grafanaClient.ListAndExportDashboards(ctx)
	if err != nil {
		return fmt.Errorf("failed to list and export dashboards: %w", err)
	}

	diffedDashboards, err := getDiffedDashboards(ctx, cfg, dashboards, log)
	if err != nil {
		return fmt.Errorf("failed to get diffed dashboards: %w", err)
	}

	if len(diffedDashboards) == 0 {
		log.Info().Msg("no changes detected in dashboards")
		return nil
	}

	if err := saveDashboards(ctx, cfg, diffedDashboards, log); err != nil {
		return fmt.Errorf("failed to save dashboards: %w", err)
	}

	if err := gitClient.CommitAll(ctx, cfg.SSHUser, cfg.SSHEmail); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	if err := gitClient.Push(ctx, branchName); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	log.Info().Str("branch", branchName).Msg("pushed changes to branch")
	return nil
}

func setupSignalHandler(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()
}

func getDiffedDashboards(ctx context.Context, cfg *config.Config, dashboards []sdk.Board, log *zerolog.Logger) ([]sdk.Board, error) {
	var diffedDashboards []sdk.Board
	for _, dashboard := range dashboards {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			dashboardJSON, err := json.MarshalIndent(dashboard, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal dashboard: %w", err)
			}

			repoDashboardPath := filepath.Join(cfg.RepoSavePath, fmt.Sprintf("%s.json", dashboard.UID))
			repoDashboardJSON, err := os.ReadFile(repoDashboardPath)
			if err != nil {
				if !os.IsNotExist(err) {
					return nil, fmt.Errorf("failed to read existing dashboard file: %w", err)
				}
				diffedDashboards = append(diffedDashboards, dashboard)
				continue
			}

			if string(dashboardJSON) != string(repoDashboardJSON) {
				diffedDashboards = append(diffedDashboards, dashboard)
			}
		}
	}

	return diffedDashboards, nil
}

func saveDashboards(ctx context.Context, cfg *config.Config, dashboards []sdk.Board, log *zerolog.Logger) error {
	if err := os.MkdirAll(cfg.RepoSavePath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	for _, dashboard := range dashboards {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			dashboardJSON, err := json.MarshalIndent(dashboard, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal dashboard: %w", err)
			}

			filePath := filepath.Join(cfg.RepoSavePath, fmt.Sprintf("%s.json", dashboard.UID))
			if err := os.WriteFile(filePath, dashboardJSON, 0644); err != nil {
				return fmt.Errorf("failed to write dashboard file: %w", err)
			}

			log.Info().
				Str("dashboard", dashboard.UID).
				Str("path", filePath).
				Msg("saved dashboard")
		}
	}
	return nil
}
