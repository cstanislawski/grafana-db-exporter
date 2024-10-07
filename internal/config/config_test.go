package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sshKeyPath := filepath.Join(tempDir, "id_rsa")
	err = os.WriteFile(sshKeyPath, []byte("dummy ssh key"), 0600)
	if err != nil {
		t.Fatalf("Failed to write dummy SSH key: %v", err)
	}

	knownHostsPath := filepath.Join(tempDir, "known_hosts")
	err = os.WriteFile(knownHostsPath, []byte("dummy known hosts"), 0600)
	if err != nil {
		t.Fatalf("Failed to write dummy known hosts: %v", err)
	}

	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
	}{
		{
			name: "Valid configuration",
			envVars: map[string]string{
				"SSH_URL":                  "git@github.com:test/repo.git",
				"SSH_KEY":                  sshKeyPath,
				"SSH_USER":                 "testuser",
				"SSH_EMAIL":                "test@example.com",
				"REPO_SAVE_PATH":           tempDir,
				"GRAFANA_URL":              "http://grafana:3000",
				"GRAFANA_SA_TOKEN":         "testtoken",
				"SSH_KNOWN_HOSTS_PATH":     knownHostsPath,
				"SSH_ACCEPT_UNKNOWN_HOSTS": "false",
				"ENABLE_RETRIES":           "true",
				"NUM_OF_RETRIES":           "5",
				"RETRIES_BACKOFF":          "10",
			},
			wantErr: false,
		},
		{
			name: "Missing required environment variable",
			envVars: map[string]string{
				"SSH_URL": "git@github.com:test/repo.git",
			},
			wantErr: true,
		},
		{
			name: "Invalid Grafana URL",
			envVars: map[string]string{
				"SSH_URL":                  "git@github.com:test/repo.git",
				"SSH_KEY":                  sshKeyPath,
				"SSH_USER":                 "testuser",
				"SSH_EMAIL":                "test@example.com",
				"REPO_SAVE_PATH":           tempDir,
				"GRAFANA_URL":              "invalid-url",
				"GRAFANA_SA_TOKEN":         "testtoken",
				"SSH_KNOWN_HOSTS_PATH":     knownHostsPath,
				"SSH_ACCEPT_UNKNOWN_HOSTS": "false",
			},
			wantErr: true,
		},
		{
			name: "Invalid NUM_OF_RETRIES",
			envVars: map[string]string{
				"SSH_URL":                  "git@github.com:test/repo.git",
				"SSH_KEY":                  sshKeyPath,
				"SSH_USER":                 "testuser",
				"SSH_EMAIL":                "test@example.com",
				"REPO_SAVE_PATH":           tempDir,
				"GRAFANA_URL":              "http://grafana:3000",
				"GRAFANA_SA_TOKEN":         "testtoken",
				"SSH_KNOWN_HOSTS_PATH":     knownHostsPath,
				"SSH_ACCEPT_UNKNOWN_HOSTS": "false",
				"NUM_OF_RETRIES":           "-1",
			},
			wantErr: true,
		},
		{
			name: "Default values for retry configuration",
			envVars: map[string]string{
				"SSH_URL":                  "git@github.com:test/repo.git",
				"SSH_KEY":                  sshKeyPath,
				"SSH_USER":                 "testuser",
				"SSH_EMAIL":                "test@example.com",
				"REPO_SAVE_PATH":           tempDir,
				"GRAFANA_URL":              "http://grafana:3000",
				"GRAFANA_SA_TOKEN":         "testtoken",
				"SSH_KNOWN_HOSTS_PATH":     knownHostsPath,
				"SSH_ACCEPT_UNKNOWN_HOSTS": "false",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()

			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cfg, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && cfg != nil {
				if tt.envVars["ENABLE_RETRIES"] == "" && !cfg.EnableRetries {
					t.Errorf("Expected default ENABLE_RETRIES to be true, got false")
				}
				if tt.envVars["NUM_OF_RETRIES"] == "" && cfg.NumOfRetries != 3 {
					t.Errorf("Expected default NUM_OF_RETRIES to be 3, got %d", cfg.NumOfRetries)
				}
				if tt.envVars["RETRIES_BACKOFF"] == "" && cfg.RetriesBackoff != 5 {
					t.Errorf("Expected default RETRIES_BACKOFF to be 5, got %d", cfg.RetriesBackoff)
				}

				if tt.envVars["NUM_OF_RETRIES"] == "5" && cfg.NumOfRetries != 5 {
					t.Errorf("Expected NUM_OF_RETRIES to be 5, got %d", cfg.NumOfRetries)
				}
				if tt.envVars["RETRIES_BACKOFF"] == "10" && cfg.RetriesBackoff != 10 {
					t.Errorf("Expected RETRIES_BACKOFF to be 10, got %d", cfg.RetriesBackoff)
				}
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sshKeyPath := filepath.Join(tempDir, "id_rsa")
	err = os.WriteFile(sshKeyPath, []byte("dummy ssh key"), 0600)
	if err != nil {
		t.Fatalf("Failed to write dummy SSH key: %v", err)
	}

	knownHostsPath := filepath.Join(tempDir, "known_hosts")
	err = os.WriteFile(knownHostsPath, []byte("dummy known hosts"), 0600)
	if err != nil {
		t.Fatalf("Failed to write dummy known hosts: %v", err)
	}

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "Valid configuration with all fields",
			cfg: &Config{
				SSHURL:                "git@github.com:test/repo.git",
				SSHKey:                sshKeyPath,
				SSHUser:               "testuser",
				SSHEmail:              "test@example.com",
				RepoSavePath:          tempDir,
				GrafanaURL:            "http://grafana:3000",
				GrafanaSaToken:        "testtoken",
				SshKnownHostsPath:     knownHostsPath,
				SshAcceptUnknownHosts: false,
				EnableRetries:         true,
				NumOfRetries:          3,
				RetriesBackoff:        5,
				LogLevel:              "info",
			},
			wantErr: false,
		},
		{
			name: "Valid configuration with minimal required fields",
			cfg: &Config{
				SSHURL:         "git@github.com:test/repo.git",
				SSHKey:         sshKeyPath,
				SSHUser:        "testuser",
				SSHEmail:       "test@example.com",
				RepoSavePath:   tempDir,
				GrafanaURL:     "http://grafana:3000",
				GrafanaSaToken: "testtoken",
				LogLevel:       "info",
			},
			wantErr: false,
		},
		{
			name: "Invalid Grafana URL",
			cfg: &Config{
				SSHURL:         "git@github.com:test/repo.git",
				SSHKey:         sshKeyPath,
				SSHUser:        "testuser",
				SSHEmail:       "test@example.com",
				RepoSavePath:   tempDir,
				GrafanaURL:     "invalid-url",
				GrafanaSaToken: "testtoken",
				LogLevel:       "info",
			},
			wantErr: true,
		},
		{
			name: "Missing SSH key file",
			cfg: &Config{
				SSHURL:         "git@github.com:test/repo.git",
				SSHKey:         "/non/existent/path",
				SSHUser:        "testuser",
				SSHEmail:       "test@example.com",
				RepoSavePath:   tempDir,
				GrafanaURL:     "http://grafana:3000",
				GrafanaSaToken: "testtoken",
				LogLevel:       "info",
			},
			wantErr: true,
		},
		{
			name: "Missing known hosts file when SshAcceptUnknownHosts is false",
			cfg: &Config{
				SSHURL:                "git@github.com:test/repo.git",
				SSHKey:                sshKeyPath,
				SSHUser:               "testuser",
				SSHEmail:              "test@example.com",
				RepoSavePath:          tempDir,
				GrafanaURL:            "http://grafana:3000",
				GrafanaSaToken:        "testtoken",
				SshKnownHostsPath:     "/non/existent/path",
				SshAcceptUnknownHosts: false,
				LogLevel:              "info",
			},
			wantErr: true,
		},
		{
			name: "Valid configuration with SshAcceptUnknownHosts true and missing known hosts file",
			cfg: &Config{
				SSHURL:                "git@github.com:test/repo.git",
				SSHKey:                sshKeyPath,
				SSHUser:               "testuser",
				SSHEmail:              "test@example.com",
				RepoSavePath:          tempDir,
				GrafanaURL:            "http://grafana:3000",
				GrafanaSaToken:        "testtoken",
				SshKnownHostsPath:     "/non/existent/path",
				SshAcceptUnknownHosts: true,
				LogLevel:              "info",
			},
			wantErr: false,
		},
		{
			name: "Invalid log level",
			cfg: &Config{
				SSHURL:         "git@github.com:test/repo.git",
				SSHKey:         sshKeyPath,
				SSHUser:        "testuser",
				SSHEmail:       "test@example.com",
				RepoSavePath:   tempDir,
				GrafanaURL:     "http://grafana:3000",
				GrafanaSaToken: "testtoken",
				LogLevel:       "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
