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

	BaseBranch            string
	BranchPrefix          string
	SshKeyPassword        string
	SshAcceptUnknownHosts bool
	SshKnownHostsPath     string
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
	"BASE_BRANCH":              "main",
	"BRANCH_PREFIX":            "grafana-db-exporter-",
	"SSH_KEY_PASSWORD":         "",
	"SSH_ACCEPT_UNKNOWN_HOSTS": "false",
}

func setDefaultsIfEmpty() {
	for envVar, defaultValue := range optionalEnvVars {
		if os.Getenv(envVar) == "" {
			os.Setenv(envVar, defaultValue)
		}
	}
}

func Load() (*Config, error) {
	setDefaultsIfEmpty()

	cfg := &Config{
		SSHURL:        os.Getenv("SSH_URL"),
		SSHKey:        os.Getenv("SSH_KEY"),
		SSHUser:       os.Getenv("SSH_USER"),
		SSHEmail:      os.Getenv("SSH_EMAIL"),
		RepoClonePath: repoClonePath,
		RepoSavePath:  repoClonePath + os.Getenv("REPO_SAVE_PATH"),
		GrafanaURL:    os.Getenv("GRAFANA_URL"),
		GrafanaApiKey: os.Getenv("GRAFANA_API_KEY"),

		BaseBranch:            os.Getenv("BASE_BRANCH"),
		BranchPrefix:          os.Getenv("BRANCH_PREFIX"),
		SshKeyPassword:        os.Getenv("SSH_KEY_PASSWORD"),
		SshAcceptUnknownHosts: os.Getenv("SSH_ACCEPT_UNKNOWN_HOSTS") == "true",
		SshKnownHostsPath:     os.Getenv("SSH_KNOWN_HOSTS_PATH"),
	}

	if missingVars := cfg.checkRequiredEnvVars(); len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missingVars)
	}

	return cfg, nil
}

func (c *Config) checkRequiredEnvVars() []string {
	var missingVars []string
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			missingVars = append(missingVars, envVar)
		}
	}

	if c.SshAcceptUnknownHosts && c.SshKnownHostsPath == "" {
		missingVars = append(missingVars, "SSH_KNOWN_HOSTS_PATH")
	}
	return missingVars
}
