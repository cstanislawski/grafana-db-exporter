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

	"grafana-db-exporter/internal/config"
	"grafana-db-exporter/internal/git"
	"grafana-db-exporter/internal/grafana"
	"grafana-db-exporter/internal/logger"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupSignalHandler(cancel)

	if err := run(ctx); err != nil {
		logger.Log.Fatal().Err(err).Msg("Application failed")
	}

	logger.Log.Info().Msg("Grafana DB exporter completed successfully")
}

func run(ctx context.Context) error {
	logger.New()
	logger.Log.Info().Msg("Starting Grafana DB exporter")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	gitClient, err := setupGitClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to setup Git client: %w", err)
	}

	grafanaClient, err := grafana.New(cfg.GrafanaURL, cfg.GrafanaSaToken)
	if err != nil {
		return fmt.Errorf("failed to create Grafana client: %w", err)
	}

	branchName, err := createNewBranch(ctx, gitClient, cfg)
	if err != nil {
		return fmt.Errorf("failed to create new branch: %w", err)
	}
	logger.Log.Info().Str("branch", branchName).Msg("Created new git branch")

	dashboards, err := fetchDashboards(ctx, grafanaClient)
	if err != nil {
		return fmt.Errorf("failed to fetch dashboards: %w", err)
	}
	logger.Log.Info().Int("count", len(dashboards)).Msg("Fetched dashboards")

	savedCount, err := saveDashboards(ctx, dashboards, cfg.RepoSavePath)
	if err != nil {
		return fmt.Errorf("failed to save dashboards: %w", err)
	}

	if savedCount > 0 {
		if err := commitAndPushChanges(ctx, gitClient, cfg, branchName); err != nil {
			return fmt.Errorf("failed to commit and push changes: %w", err)
		}
		logger.Log.Info().Int("count", savedCount).Str("branch", branchName).Msg("Committed and pushed dashboard changes")
	} else {
		logger.Log.Info().Msg("No changes to commit")
	}

	return nil
}

func setupGitClient(cfg *config.Config) (*git.Client, error) {
	return git.New(cfg.RepoClonePath, cfg.SSHURL, cfg.SSHKey, cfg.SshKeyPassword, cfg.SshKnownHostsPath, cfg.SshAcceptUnknownHosts)
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
			if err := saveDashboard(dashboard, savePath); err != nil {
				return savedCount, fmt.Errorf("failed to save dashboard %s: %w", dashboard.UID, err)
			}
			savedCount++
		}
	}
	return savedCount, nil
}

func saveDashboard(dashboard grafana.Dashboard, savePath string) error {
	filePath := filepath.Join(savePath, fmt.Sprintf("%s.json", dashboard.UID))

	data, err := json.MarshalIndent(dashboard.Data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dashboard data: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write dashboard file: %w", err)
	}

	return nil
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
