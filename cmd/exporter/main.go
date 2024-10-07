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
	"grafana-db-exporter/internal/utils"
)

func main() {
	if err := logger.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Log.Info().Msg("Starting Grafana DB exporter")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupSignalHandler(cancel)

	if err := run(ctx); err != nil {
		logger.Log.Fatal().Err(err).Msg("Application failed")
	}

	logger.Log.Info().Msg("Grafana DB exporter completed successfully")
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	logger.Log.Debug().Interface("config", cfg).Msg("Configuration loaded")

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

	dashboards, err := utils.Retry(ctx, cfg, "fetch dashboards", func() ([]grafana.Dashboard, error) {
		return fetchDashboards(ctx, grafanaClient)
	})
	if err != nil {
		return err
	}
	logger.Log.Info().Int("count", len(dashboards)).Msg("Fetched dashboards")

	savedCount, err := utils.Retry(ctx, cfg, "save dashboards", func() (int, error) {
		return saveDashboards(ctx, dashboards, cfg.RepoSavePath)
	})
	if err != nil {
		return err
	}
	logger.Log.Debug().Int("count", savedCount).Msg("Saved dashboards")

	if savedCount > 0 {
		_, err = utils.Retry(ctx, cfg, "commit and push changes", func() (interface{}, error) {
			return nil, commitAndPushChanges(ctx, gitClient, cfg, branchName)
		})
		if err != nil {
			return err
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
	logger.Log.Debug().Str("baseBranch", cfg.BaseBranch).Str("branchPrefix", cfg.BranchPrefix).Msg("Creating new branch")
	branchName := fmt.Sprintf("%s%s", cfg.BranchPrefix, time.Now().Format("20060102150405"))
	return gitClient.CheckoutNewBranch(ctx, cfg.BaseBranch, branchName)
}

func fetchDashboards(ctx context.Context, grafanaClient *grafana.Client) ([]grafana.Dashboard, error) {
	logger.Log.Debug().Msg("Fetching dashboards from Grafana")
	return grafanaClient.ListAndExportDashboards(ctx)
}

func saveDashboards(ctx context.Context, dashboards []grafana.Dashboard, savePath string) (int, error) {
	logger.Log.Debug().Int("dashboardCount", len(dashboards)).Str("savePath", savePath).Msg("Saving dashboards")
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
			logger.Log.Debug().Str("dashboardUID", dashboard.UID).Msg("Dashboard saved")
		}
	}
	return savedCount, nil
}

func saveDashboard(dashboard grafana.Dashboard, savePath string) error {
	filePath := filepath.Join(savePath, fmt.Sprintf("%s.json", dashboard.UID))
	logger.Log.Debug().Str("filePath", filePath).Msg("Saving dashboard to file")

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
	logger.Log.Debug().Str("branch", branchName).Msg("Committing changes")
	if err := gitClient.CommitAll(ctx, cfg.SSHUser, cfg.SSHEmail); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	logger.Log.Debug().Str("branch", branchName).Msg("Pushing changes")
	if err := gitClient.Push(ctx, branchName); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}

func setupSignalHandler(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Log.Debug().Str("signal", sig.String()).Msg("Received termination signal")
		cancel()
	}()
}
