package grafana

import (
	"context"
	"fmt"

	"grafana-db-exporter/internal/logger"

	"github.com/grafana-tools/sdk"
)

type Dashboard struct {
	UID   string
	Title string
	Data  interface{}
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
	boardLinks, err := gc.client.SearchDashboards(ctx, "", false)
	if err != nil {
		return nil, fmt.Errorf("failed to search dashboards: %w", err)
	}
	logger.Log.Debug().Int("dashboardCount", len(boardLinks)).Msg("Retrieved dashboard links")

	var dashboards []Dashboard
	for _, link := range boardLinks {
		logger.Log.Debug().Str("dashboardUID", link.UID).Msg("Fetching dashboard")
		board, _, err := gc.client.GetDashboardByUID(ctx, link.UID)
		if err != nil {
			return nil, fmt.Errorf("failed to get dashboard by UID: %w", err)
		}
		logger.Log.Debug().Str("dashboardUID", link.UID).Str("title", board.Title).Msg("Dashboard retrieved")

		dashboards = append(dashboards, Dashboard{
			UID:   board.UID,
			Title: board.Title,
			Data:  board,
		})
	}

	logger.Log.Debug().Int("exportedDashboards", len(dashboards)).Msg("Completed dashboard list and export operation")
	return dashboards, nil
}
