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
	RepoClonePath string
	RepoSavePath  string
	GrafanaURL    string
	GrafanaApiKey string

	BaseBranch   string
	BranchPrefix string
	OrgID        string
}

const repoClonePath = "./repo/"

var requiredEnvVars = []string{
	"SSH_URL",
	"SSH_KEY",
	"SSH_USER",
	"SSH_EMAIL",
	"REPO_SAVE_PATH",
	"GRAFANA_URL",
	"GRAFANA_API_KEY",
}

var optionalEnvVars = map[string]string{
	"BASE_BRANCH":   "main",
	"BRANCH_PREFIX": "grafana-db-exporter-",
}

func Load() (*Config, error) {
	cfg := &Config{
		SSHURL:        os.Getenv("SSH_URL"),
		SSHKey:        os.Getenv("SSH_KEY"),
		SSHUser:       os.Getenv("SSH_USER"),
		SSHEmail:      os.Getenv("SSH_EMAIL"),
		BaseBranch:    os.Getenv("BASE_BRANCH"),
		RepoClonePath: repoClonePath,
		RepoSavePath:  repoClonePath + os.Getenv("REPO_SAVE_PATH"),
		BranchPrefix:  os.Getenv("BRANCH_PREFIX"),
		GrafanaURL:    os.Getenv("GRAFANA_URL"),
		GrafanaApiKey: os.Getenv("GRAFANA_API_KEY"),
	}

	if missingVars := cfg.checkRequiredEnvVars(); len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missingVars)
	}

	cfg.setDefaultsIfEmpty()

	return cfg, nil
}

func (c *Config) checkRequiredEnvVars() []string {
	var missingVars []string
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			missingVars = append(missingVars, envVar)
		}
	}
	return missingVars
}

func (c *Config) setDefaultsIfEmpty() {
	for envVar, defaultValue := range optionalEnvVars {
		if os.Getenv(envVar) == "" {
			os.Setenv(envVar, defaultValue)
		}
	}
}
