package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	orgID      uint
}

type Dashboard struct {
	UID      string
	Data     json.RawMessage
	FolderID int64
	IsFolder bool
}

func New(baseURL, apiKey string, orgID uint) (*Client, error) {
	_, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Grafana URL: %w", err)
	}

	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: http.DefaultClient,
		orgID:      orgID,
	}, nil
}

func (c *Client) ListAndExportDashboards(ctx context.Context) ([]Dashboard, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/search?type=dash-db", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.addHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var searchResult []struct {
		UID   string `json:"uid"`
		Title string `json:"title"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var dashboards []Dashboard
	for _, result := range searchResult {
		dashboard, err := c.getDashboard(ctx, result.UID)
		if err != nil {
			return nil, fmt.Errorf("getting dashboard %s: %w", result.UID, err)
		}
		dashboards = append(dashboards, dashboard)
	}

	return dashboards, nil
}

func (c *Client) getDashboard(ctx context.Context, uid string) (Dashboard, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/dashboards/uid/"+uid, nil)
	if err != nil {
		return Dashboard{}, fmt.Errorf("creating request: %w", err)
	}

	c.addHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Dashboard{}, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Dashboard{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var dashboardWrapper struct {
		Dashboard json.RawMessage `json:"dashboard"`
		Meta      struct {
			IsFolder bool   `json:"isFolder"`
			Folder   int64  `json:"folderId"`
			UID      string `json:"uid"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dashboardWrapper); err != nil {
		return Dashboard{}, fmt.Errorf("decoding response: %w", err)
	}

	return Dashboard{
		UID:      dashboardWrapper.Meta.UID,
		Data:     dashboardWrapper.Dashboard,
		FolderID: dashboardWrapper.Meta.Folder,
		IsFolder: dashboardWrapper.Meta.IsFolder,
	}, nil
}

func (c *Client) addHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Grafana-Org-Id", strconv.FormatUint(uint64(c.orgID), 10))
}
