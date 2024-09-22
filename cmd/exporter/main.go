package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"grafana-db-exporter/pkg/config"
	"grafana-db-exporter/pkg/git"
	"grafana-db-exporter/pkg/grafana"
	"grafana-db-exporter/pkg/logger"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupSignalHandler(cancel)

	logger.New()

	logger.Log.Info().Msg("starting grafana-db-exporter")

	cfg, err := config.Load()
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to load configuration")
	}

	if err := run(ctx, cfg); err != nil {
		logger.Log.Fatal().Err(err).Msg("application failed")
	}

	logger.Log.Info().Msg("grafana-db-exporter completed successfully")
}

func run(ctx context.Context, cfg *config.Config) error {
	gitClient, err := git.New(cfg.RepoClonePath, cfg.SSHURL, cfg.SSHKey, cfg.SshKeyPassword, cfg.SshKnownHostsPath, cfg.SshAcceptUnknownHosts)
	if err != nil {
		return fmt.Errorf("failed to create Git client: %w", err)
	}

	grafanaClient, err := grafana.New(cfg.GrafanaURL, cfg.GrafanaSaToken)
	if err != nil {
		return fmt.Errorf("failed to create Grafana client: %w", err)
	}

	branchName, err := createNewBranch(ctx, gitClient, cfg)
	if err != nil {
		return fmt.Errorf("failed to create new branch: %w", err)
	}
	logger.Log.Info().Str("branch", branchName).Msg("created new git branch")

	dashboards, err := fetchDashboards(ctx, grafanaClient)
	if err != nil {
		return fmt.Errorf("failed to fetch dashboards: %w", err)
	}
	logger.Log.Info().Int("count", len(dashboards)).Msg("fetched dashboards")

	savedCount, err := saveDashboards(ctx, dashboards, cfg.RepoSavePath)
	if err != nil {
		return fmt.Errorf("failed to save dashboards: %w", err)
	}

	if savedCount > 0 {
		if err := commitAndPushChanges(ctx, gitClient, cfg, branchName); err != nil {
			return fmt.Errorf("failed to commit and push changes: %w", err)
		}
		logger.Log.Info().Int("count", savedCount).Str("branch", branchName).Msg("committed and pushed dashboard changes")
	} else {
		logger.Log.Info().Msg("no changes to commit")
	}

	return nil
}

func createNewBranch(ctx context.Context, gitClient *git.Client, cfg *config.Config) (string, error) {
	branchName := fmt.Sprintf("%s%s", cfg.BranchPrefix, time.Now().Format("20060102150405"))
	return gitClient.CheckoutNewBranch(ctx, cfg.BaseBranch, branchName)
}

func fetchDashboards(ctx context.Context, grafanaClient *grafana.Client) ([]grafana.Dashboard, error) {
	return grafanaClient.ListAndExportDashboards(ctx)
}

func saveDashboards(ctx context.Context, dashboards []grafana.Dashboard, savePath string) (int, error) {
	savedCount := 0
	for _, dashboard := range dashboards {
		select {
		case <-ctx.Done():
			return savedCount, ctx.Err()
		default:
			dashboardJSON, err := json.MarshalIndent(dashboard, "", "  ")
			if err != nil {
				return savedCount, fmt.Errorf("failed to marshal dashboard: %w", err)
			}

			filePath := filepath.Join(savePath, fmt.Sprintf("%s.json", dashboard.UID))
			if err := os.WriteFile(filePath, dashboardJSON, 0644); err != nil {
				return savedCount, fmt.Errorf("failed to write dashboard file: %w", err)
			}
			savedCount++
		}
	}
	return savedCount, nil
}

func commitAndPushChanges(ctx context.Context, gitClient *git.Client, cfg *config.Config, branchName string) error {
	if err := gitClient.CommitAll(ctx, cfg.SSHUser, cfg.SSHEmail); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	if err := gitClient.Push(ctx, branchName); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

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
