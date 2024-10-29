package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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

	cfg, err := config.Load()
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	if err := run(ctx, cfg); err != nil {
		logger.Log.Fatal().Err(err).Msg("Application failed")
	}

	logger.Log.Info().Msg("Grafana DB exporter completed successfully")
}

func run(ctx context.Context, cfg *config.Config) error {
	switch cfg.RunMode {
	case config.OneTime:
		return runOnce(ctx, cfg)
	case config.Periodic:
		return runPeriodic(ctx, cfg)
	default:
		return fmt.Errorf("unknown run mode: %s", cfg.RunMode)
	}
}
func runPeriodic(ctx context.Context, cfg *config.Config) error {
	ticker := time.NewTicker(cfg.SyncInterval)
	defer ticker.Stop()

	branchManager := newBranchManager(cfg)

	if err := executeSync(ctx, cfg, branchManager); err != nil {
		logger.Log.Error().Err(err).Msg("Initial sync failed")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := executeSync(ctx, cfg, branchManager); err != nil {
				logger.Log.Error().Err(err).Msg("Periodic sync failed")
			}
		}
	}
}

func runOnce(ctx context.Context, cfg *config.Config) error {
	return executeSync(ctx, cfg, newBranchManager(cfg))
}

type branchManager struct {
	cfg           *config.Config
	currentBranch string
	createdAt     time.Time
	changeCount   int
}

func newBranchManager(cfg *config.Config) *branchManager {
	return &branchManager{
		cfg: cfg,
	}
}

func (bm *branchManager) getBranchName() string {
	if bm.cfg.BranchStrategy == config.NewBranch ||
		bm.currentBranch == "" ||
		(bm.cfg.BranchStrategy == config.ReuseBranch && time.Since(bm.createdAt) > bm.cfg.BranchTTL) {
		bm.currentBranch = fmt.Sprintf("%s%s", bm.cfg.BranchPrefix, time.Now().Format("20060102150405"))
		bm.createdAt = time.Now()
		bm.changeCount = 0
	}
	return bm.currentBranch
}

func executeSync(ctx context.Context, cfg *config.Config, bm *branchManager) error {
	gitClient, err := setupGitClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to setup Git client: %w", err)
	}

	grafanaClient, err := grafana.New(cfg.GrafanaURL, cfg.GrafanaSaToken)
	if err != nil {
		return fmt.Errorf("failed to create Grafana client: %w", err)
	}

	branchName := bm.getBranchName()
	if branchName != bm.currentBranch {
		logger.Log.Info().
			Str("oldBranch", bm.currentBranch).
			Str("newBranch", branchName).
			Msg("Creating new branch")
	}

	if err := checkoutBranch(ctx, gitClient, cfg, branchName); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	dashboards, err := utils.Retry(ctx, cfg, "fetch dashboards", func() ([]grafana.Dashboard, error) {
		return fetchDashboards(ctx, grafanaClient)
	})
	if err != nil {
		return err
	}
	logger.Log.Info().Int("count", len(dashboards)).Msg("Fetched dashboards")

	if cfg.DeleteMissing {
		if err := deleteMissingDashboards(cfg.RepoSavePath, dashboards, cfg); err != nil {
			return fmt.Errorf("failed to delete missing dashboards: %w", err)
		}
	}

	savedCount, err := utils.Retry(ctx, cfg, "save dashboards", func() (int, error) {
		return saveDashboards(ctx, dashboards, cfg)
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
		logger.Log.Info().
			Int("count", savedCount).
			Str("branch", branchName).
			Msg("Committed and pushed dashboard changes")
		bm.changeCount += savedCount
	} else {
		logger.Log.Info().Msg("No changes to commit")
	}

	return nil
}

func checkoutBranch(ctx context.Context, gitClient *git.Client, cfg *config.Config, branchName string) error {
	exists, err := gitClient.BranchExists(ctx, branchName)
	if err != nil {
		return fmt.Errorf("failed to check branch existence: %w", err)
	}

	if exists {
		err = gitClient.CheckoutBranch(ctx, branchName)
	} else {
		_, err = gitClient.CheckoutNewBranch(ctx, cfg.BaseBranch, branchName)
	}

	if err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	return nil
}

func deleteMissingDashboards(repoSavePath string, fetchedDashboards []grafana.Dashboard, cfg *config.Config) error {
	existingFiles := make(map[string]bool)
	fetchedPaths := make(map[string]bool)

	err := filepath.Walk(repoSavePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".json") {
			relPath, err := filepath.Rel(repoSavePath, path)
			if err != nil {
				return fmt.Errorf("failed to get relative path: %w", err)
			}
			existingFiles[relPath] = true
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk repository directory: %w", err)
	}

	for _, dashboard := range fetchedDashboards {
		relPath, err := filepath.Rel(
			repoSavePath,
			grafana.GetDashboardPath(repoSavePath, dashboard, cfg.IgnoreFolderStructure),
		)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		fetchedPaths[relPath] = true
	}

	for existingPath := range existingFiles {
		if !fetchedPaths[existingPath] {
			fullPath := filepath.Join(repoSavePath, existingPath)
			if err := os.Remove(fullPath); err != nil {
				return fmt.Errorf("failed to delete file %s: %w", existingPath, err)
			}
			logger.Log.Info().Str("file", existingPath).Msg("Deleted missing dashboard file")
		}
	}

	if cfg.IgnoreFolderStructure {
		return nil
	}

	err = filepath.Walk(repoSavePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == repoSavePath {
			return nil
		}
		if info.IsDir() {
			empty, err := isDirEmpty(path)
			if err != nil {
				return fmt.Errorf("failed to check if directory is empty: %w", err)
			}
			if empty {
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("failed to remove empty directory %s: %w", path, err)
				}
				logger.Log.Info().Str("directory", path).Msg("Removed empty directory")
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to clean up empty directories: %w", err)
	}

	return nil
}

func isDirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
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

func saveDashboards(ctx context.Context, dashboards []grafana.Dashboard, cfg *config.Config) (int, error) {
	logger.Log.Debug().
		Int("dashboardCount", len(dashboards)).
		Str("savePath", cfg.RepoSavePath).
		Bool("ignoreFolderStructure", cfg.IgnoreFolderStructure).
		Msg("Saving dashboards")

	savedCount := 0
	for _, dashboard := range dashboards {
		select {
		case <-ctx.Done():
			return savedCount, ctx.Err()
		default:
			fullPath := grafana.GetDashboardPath(cfg.RepoSavePath, dashboard, cfg.IgnoreFolderStructure)
			dirPath := filepath.Dir(fullPath)

			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return savedCount, fmt.Errorf("failed to create directory %s: %w", dirPath, err)
			}

			if err := saveDashboard(dashboard, fullPath, cfg); err != nil {
				return savedCount, fmt.Errorf("failed to save dashboard %s: %w", dashboard.UID, err)
			}
			savedCount++
			logger.Log.Debug().
				Str("dashboardUID", dashboard.UID).
				Str("folder", dashboard.FolderTitle).
				Bool("ignoredFolderStructure", cfg.IgnoreFolderStructure).
				Msg("Dashboard saved")
		}
	}
	return savedCount, nil
}

func saveDashboard(dashboard grafana.Dashboard, filePath string, cfg *config.Config) error {
	logger.Log.Debug().Str("filePath", filePath).Msg("Saving dashboard to file")

	data, err := json.MarshalIndent(dashboard.Data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dashboard data: %w", err)
	}

	if cfg.AddMissingNewlines && len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
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

	if !cfg.DryRun {
		logger.Log.Debug().Str("branch", branchName).Msg("Pushing changes")
		if err := gitClient.Push(ctx, branchName); err != nil {
			return fmt.Errorf("failed to push changes: %w", err)
		}
	} else {
		logger.Log.Info().Msg("Dry run mode: Changes committed but not pushed")
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
