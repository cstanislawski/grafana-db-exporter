package grafana

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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
		folderResponse string
		searchResponse string
		dashResponse   string
		wantErr        bool
		wantLen        int
		checkFolders   bool
	}{
		{
			name:           "Valid dashboards with folders",
			folderResponse: `[{"id":1,"uid":"folder1","title":"Test Folder"}]`,
			searchResponse: `[{"uid":"dash1","folderID":1},{"uid":"dash2","folderID":0}]`,
			dashResponse:   `{"dashboard":{"uid":"dash1","title":"Test Dashboard"}}`,
			wantErr:        false,
			wantLen:        2,
			checkFolders:   true,
		},
		{
			name:           "Empty dashboard list",
			folderResponse: `[]`,
			searchResponse: `[]`,
			wantErr:        false,
			wantLen:        0,
			checkFolders:   false,
		},
		{
			name:           "Invalid folder response",
			folderResponse: `invalid json`,
			searchResponse: `[]`,
			wantErr:        true,
			checkFolders:   false,
		},
		{
			name:           "Invalid search response",
			folderResponse: `[]`,
			searchResponse: `invalid json`,
			wantErr:        true,
			checkFolders:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var err error
				w.Header().Set("Content-Type", "application/json")

				switch r.URL.Path {
				case "/api/folders":
					_, err = w.Write([]byte(tt.folderResponse))
				case "/api/search":
					_, err = w.Write([]byte(tt.searchResponse))
				case "/api/dashboards/uid/dash1", "/api/dashboards/uid/dash2":
					_, err = w.Write([]byte(tt.dashResponse))
				default:
					http.Error(w, "Not found", http.StatusNotFound)
					return
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

			if tt.checkFolders && !tt.wantErr && len(boards) > 0 {
				foundFoldered := false
				foundRoot := false
				for _, board := range boards {
					if board.FolderID == 1 {
						foundFoldered = true
						if board.FolderTitle != "Test Folder" {
							t.Errorf("Expected folder title 'Test Folder', got %s", board.FolderTitle)
						}
					}
					if board.FolderID == 0 {
						foundRoot = true
						if board.FolderTitle != "" {
							t.Errorf("Expected empty folder title for root dashboard, got %s", board.FolderTitle)
						}
					}
				}
				if !foundFoldered || !foundRoot {
					t.Error("Did not find expected combination of foldered and root dashboards")
				}
			}
		})
	}
}

func TestSanitizeFolderPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Normal path",
			path:     "Test Folder",
			expected: "Test Folder",
		},
		{
			name:     "Path with invalid characters",
			path:     `Test/Folder:with*invalid?chars`,
			expected: "Test-Folder-with-invalid-chars",
		},
		{
			name:     "Path with spaces and trim",
			path:     "  Test  Folder  ",
			expected: "Test  Folder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFolderPath(tt.path)
			if result != tt.expected {
				t.Errorf("SanitizeFolderPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetDashboardPath(t *testing.T) {
	tests := []struct {
		name                  string
		basePath              string
		dashboard             Dashboard
		ignoreFolderStructure bool
		expected              string
	}{
		{
			name:     "Root dashboard",
			basePath: "/base/path",
			dashboard: Dashboard{
				UID:      "dash1",
				FolderID: 0,
			},
			ignoreFolderStructure: false,
			expected:              filepath.Join("/base/path", "dash1.json"),
		},
		{
			name:     "Foldered dashboard",
			basePath: "/base/path",
			dashboard: Dashboard{
				UID:         "dash2",
				FolderID:    1,
				FolderTitle: "Test Folder",
			},
			ignoreFolderStructure: false,
			expected:              filepath.Join("/base/path", "Test Folder", "dash2.json"),
		},
		{
			name:     "Foldered dashboard with ignore structure",
			basePath: "/base/path",
			dashboard: Dashboard{
				UID:         "dash2",
				FolderID:    1,
				FolderTitle: "Test Folder",
			},
			ignoreFolderStructure: true,
			expected:              filepath.Join("/base/path", "dash2.json"),
		},
		{
			name:     "Dashboard with sanitized folder path",
			basePath: "/base/path",
			dashboard: Dashboard{
				UID:         "dash3",
				FolderID:    1,
				FolderTitle: "Test/Folder:with*invalid?chars",
			},
			ignoreFolderStructure: false,
			expected:              filepath.Join("/base/path", "Test-Folder-with-invalid-chars", "dash3.json"),
		},
		{
			name:     "Dashboard with sanitized folder path and ignore structure",
			basePath: "/base/path",
			dashboard: Dashboard{
				UID:         "dash3",
				FolderID:    1,
				FolderTitle: "Test/Folder:with*invalid?chars",
			},
			ignoreFolderStructure: true,
			expected:              filepath.Join("/base/path", "dash3.json"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDashboardPath(tt.basePath, tt.dashboard, tt.ignoreFolderStructure)
			if result != tt.expected {
				t.Errorf("GetDashboardPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}
