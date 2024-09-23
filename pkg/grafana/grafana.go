package grafana

import (
	"context"
	"fmt"

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
	client, err := sdk.NewClient(url, apiKey, sdk.DefaultHTTPClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create Grafana client: %w", err)
	}
	return &Client{client: client}, nil
}

func (gc *Client) ListAndExportDashboards(ctx context.Context) ([]Dashboard, error) {
	boardLinks, err := gc.client.SearchDashboards(ctx, "", false)
	if err != nil {
		return nil, fmt.Errorf("failed to search dashboards: %w", err)
	}

	var dashboards []Dashboard
	for _, link := range boardLinks {
		board, _, err := gc.client.GetDashboardByUID(ctx, link.UID)
		if err != nil {
			return nil, fmt.Errorf("failed to get dashboard by UID: %w", err)
		}
		dashboards = append(dashboards, Dashboard{
			UID:   board.UID,
			Title: board.Title,
			Data:  board,
		})
	}

	return dashboards, nil
}
