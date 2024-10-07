package grafana

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		apiKey  string
		wantErr bool
	}{
		{
			name:    "Valid client creation",
			url:     "http://grafana:3000",
			apiKey:  "validapikey",
			wantErr: false,
		},
		{
			name:    "Invalid URL",
			url:     "://invalid-url",
			apiKey:  "validapikey",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.url, tt.apiKey)
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
		wantErr        bool
		wantLen        int
	}{
		{
			name:           "Valid dashboards",
			searchResponse: `[{"uid":"dash1"},{"uid":"dash2"}]`,
			dashResponse:   `{"dashboard":{"uid":"dash1","title":"Test Dashboard"}}`,
			wantErr:        false,
			wantLen:        2,
		},
		{
			name:           "Empty dashboard list",
			searchResponse: `[]`,
			wantErr:        false,
			wantLen:        0,
		},
		{
			name:           "Invalid search response",
			searchResponse: `invalid json`,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var err error
				switch r.URL.Path {
				case "/api/search":
					_, err = w.Write([]byte(tt.searchResponse))
				case "/api/dashboards/uid/dash1", "/api/dashboards/uid/dash2":
					_, err = w.Write([]byte(tt.dashResponse))
				}
				if err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			client, _ := New(server.URL, "testkey")
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
