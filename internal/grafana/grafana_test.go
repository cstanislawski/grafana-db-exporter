package grafana

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		apiKey  string
		orgID   uint
		wantErr bool
	}{
		{
			name:    "Valid client creation",
			url:     "http://grafana:3000",
			apiKey:  "validapikey",
			orgID:   1,
			wantErr: false,
		},
		{
			name:    "Invalid URL",
			url:     "://invalid-url",
			apiKey:  "validapikey",
			orgID:   1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.url, tt.apiKey, tt.orgID)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_ListAndExportDashboards(t *testing.T) {
	tests := []struct {
		name           string
		searchResponse string
		dashResponse   string
		orgID          uint
		wantErr        bool
		wantLen        int
	}{
		{
			name:           "Valid dashboards",
			searchResponse: `[{"uid":"dash1"},{"uid":"dash2"}]`,
			dashResponse:   `{"dashboard":{"uid":"dash1","title":"Test Dashboard"}}`,
			orgID:          1,
			wantErr:        false,
			wantLen:        2,
		},
		{
			name:           "Empty dashboard list",
			searchResponse: `[]`,
			orgID:          1,
			wantErr:        false,
			wantLen:        0,
		},
		{
			name:           "Invalid search response",
			searchResponse: `invalid json`,
			orgID:          1,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				orgIDHeader := r.Header.Get("X-Grafana-Org-Id")
				if orgIDHeader != strconv.FormatUint(uint64(tt.orgID), 10) {
					t.Errorf("Expected X-Grafana-Org-Id header to be %d, got %s", tt.orgID, orgIDHeader)
				}

				switch r.URL.Path {
				case "/api/search":
					w.Write([]byte(tt.searchResponse))
				case "/api/dashboards/uid/dash1", "/api/dashboards/uid/dash2":
					w.Write([]byte(tt.dashResponse))
				}
			}))
			defer server.Close()

			client, _ := New(server.URL, "testkey", tt.orgID)
			boards, err := client.ListAndExportDashboards(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("Client.ListAndExportDashboards() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(boards) != tt.wantLen {
				t.Errorf("Client.ListAndExportDashboards() got %v dashboards, want %v", len(boards), tt.wantLen)
			}
		})
	}
}

func TestClient_GetDashboard(t *testing.T) {
	tests := []struct {
		name         string
		dashResponse string
		orgID        uint
		wantErr      bool
	}{
		{
			name:         "Valid dashboard",
			dashResponse: `{"dashboard":{"uid":"dash1","title":"Test Dashboard"},"meta":{"isFolder":false,"folderId":0}}`,
			orgID:        1,
			wantErr:      false,
		},
		{
			name:         "Invalid JSON response",
			dashResponse: `invalid json`,
			orgID:        1,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				orgIDHeader := r.Header.Get("X-Grafana-Org-Id")
				if orgIDHeader != strconv.FormatUint(uint64(tt.orgID), 10) {
					t.Errorf("Expected X-Grafana-Org-Id header to be %d, got %s", tt.orgID, orgIDHeader)
				}

				w.Write([]byte(tt.dashResponse))
			}))
			defer server.Close()

			client, _ := New(server.URL, "testkey", tt.orgID)
			_, err := client.getDashboard(context.Background(), "dash1")

			if (err != nil) != tt.wantErr {
				t.Errorf("Client.getDashboard() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
