package grafana

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"grafana-db-exporter/internal/logger"

	"github.com/grafana-tools/sdk"
)

type Dashboard struct {
	UID         string
	Title       string
	FolderID    int
	FolderTitle string
	Data        interface{}
}

type Client struct {
	client *sdk.Client
}

func New(url, apiKey string) (*Client, error) {
	logger.Log.Debug().Str("url", url).Msg("Creating new Grafana client")
	client, err := sdk.NewClient(url, apiKey, sdk.DefaultHTTPClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create Grafana client: %w", err)
	}
	return &Client{client: client}, nil
}

func (gc *Client) ListAndExportDashboards(ctx context.Context) ([]Dashboard, error) {
	logger.Log.Debug().Msg("Starting dashboard list and export operation")

	folders, err := gc.client.GetAllFolders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch folders: %w", err)
	}

	folderMap := make(map[int]string)
	for _, folder := range folders {
		folderMap[folder.ID] = folder.Title
	}

	boardLinks, err := gc.client.SearchDashboards(ctx, "", false)
	if err != nil {
		return nil, fmt.Errorf("failed to search dashboards: %w", err)
	}
	logger.Log.Debug().Int("dashboardCount", len(boardLinks)).Msg("Retrieved dashboard links")

	var dashboards []Dashboard
	for _, link := range boardLinks {
		logger.Log.Debug().
			Str("dashboardUID", link.UID).
			Int("folderID", link.FolderID).
			Msg("Fetching dashboard")

		board, _, err := gc.client.GetDashboardByUID(ctx, link.UID)
		if err != nil {
			return nil, fmt.Errorf("failed to get dashboard by UID: %w", err)
		}

		folderTitle := "General"
		if link.FolderID != 0 {
			var ok bool
			folderTitle, ok = folderMap[link.FolderID]
			if !ok {
				logger.Log.Warn().
					Int("folderID", link.FolderID).
					Str("dashboardUID", link.UID).
					Msg("Folder not found, using ID as name")
				folderTitle = fmt.Sprintf("folder-%d", link.FolderID)
			}
		}

		dashboards = append(dashboards, Dashboard{
			UID:         board.UID,
			Title:       board.Title,
			FolderID:    link.FolderID,
			FolderTitle: folderTitle,
			Data:        board,
		})

		logger.Log.Debug().
			Str("dashboardUID", link.UID).
			Str("title", board.Title).
			Str("folder", folderTitle).
			Msg("Dashboard retrieved")
	}

	logger.Log.Debug().Int("exportedDashboards", len(dashboards)).Msg("Completed dashboard list and export operation")
	return dashboards, nil
}

func SanitizeFolderPath(path string) string {
	invalidChars := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
	sanitized := invalidChars.ReplaceAllString(path, "-")
	return strings.TrimSpace(sanitized)
}

func GetDashboardPath(basePath string, dashboard Dashboard) string {
	folderPath := SanitizeFolderPath(dashboard.FolderTitle)
	return filepath.Join(basePath, folderPath, fmt.Sprintf("%s.json", dashboard.UID))
}
