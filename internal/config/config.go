// internal/config/config.go
package config

import (
	"fmt"
	"os"
)

type Config struct {
	SSHURL        string
	SSHKey        string
	SSHUser       string
	SSHEmail      string
	BaseBranch    string
	SavePath      string
	BranchPrefix  string
	GrafanaURL    string
	GrafanaAPIKey string // New field for API key
}

func Load() (*Config, error) {
	cfg := &Config{
		SSHURL:        os.Getenv("SSH_URL"),
		SSHKey:        os.Getenv("SSH_KEY"),
		SSHUser:       os.Getenv("SSH_USER"),
		SSHEmail:      os.Getenv("SSH_EMAIL"),
		BaseBranch:    os.Getenv("BASE_BRANCH"),
		SavePath:      os.Getenv("SAVE_PATH"),
		BranchPrefix:  os.Getenv("BRANCH_PREFIX"),
		GrafanaURL:    os.Getenv("GRAFANA_URL"),
		GrafanaAPIKey: os.Getenv("GRAFANA_API_KEY"),
	}

	if cfg.SSHURL == "" || cfg.SSHKey == "" || cfg.SSHUser == "" || cfg.SSHEmail == "" || cfg.BaseBranch == "" || cfg.SavePath == "" || cfg.GrafanaURL == "" || cfg.GrafanaAPIKey == "" {
		return nil, fmt.Errorf("missing required environment variables")
	}

	if cfg.BranchPrefix == "" {
		cfg.BranchPrefix = "grafana_db_export_"
	}

	return cfg, nil
}
