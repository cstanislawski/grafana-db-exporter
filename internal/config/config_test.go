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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			_, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
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
			name: "Valid configuration",
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
			},
			wantErr: false,
		},
		{
			name: "Invalid Grafana URL",
			cfg: &Config{
				SSHURL:                "git@github.com:test/repo.git",
				SSHKey:                sshKeyPath,
				SSHUser:               "testuser",
				SSHEmail:              "test@example.com",
				RepoSavePath:          tempDir,
				GrafanaURL:            "invalid-url",
				GrafanaSaToken:        "testtoken",
				SshKnownHostsPath:     knownHostsPath,
				SshAcceptUnknownHosts: false,
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
