// internal/grafana/grafana.go
package grafana

import (
	"context"
	"fmt"

	"github.com/grafana-tools/sdk"
)

type GrafanaClient struct {
	client *sdk.Client
}

func NewClient(url, apiKey string) (*GrafanaClient, error) {
	client, err := sdk.NewClient(url, apiKey, sdk.DefaultHTTPClient) // Use apiKey here
	if err != nil {
		return nil, fmt.Errorf("failed to create Grafana client: %w", err)
	}
	return &GrafanaClient{client: client}, nil
}

func (gc *GrafanaClient) ListAndExportDashboards(ctx context.Context) ([]sdk.Board, error) {
	boardLinks, err := gc.client.SearchDashboards(ctx, "", false)
	if err != nil {
		return nil, fmt.Errorf("failed to search dashboards: %w", err)
	}

	var boards []sdk.Board
	for _, link := range boardLinks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			board, _, err := gc.client.GetDashboardByUID(ctx, link.UID)
			if err != nil {
				return nil, fmt.Errorf("failed to get dashboard %s: %w", link.UID, err)
			}
			boards = append(boards, board)
		}
	}

	return boards, nil
}
