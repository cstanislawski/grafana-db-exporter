package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"grafana-db-exporter/internal/logger"
)

type RunMode string
type BranchStrategy string

const (
	OneTime  RunMode = "one-time"
	Periodic RunMode = "periodic"

	NewBranch   BranchStrategy = "new-branch"
	ReuseBranch BranchStrategy = "reuse-branch"
)

func (m RunMode) IsValid() bool {
	switch m {
	case OneTime, Periodic:
		return true
	default:
		return false
	}
}

func (s BranchStrategy) IsValid() bool {
	switch s {
	case NewBranch, ReuseBranch:
		return true
	default:
		return false
	}
}

type Config struct {
	SSHURL         string `env:"SSH_URL,required"`
	SSHKey         string `env:"SSH_KEY,required"`
	SSHUser        string `env:"SSH_USER,required"`
	SSHEmail       string `env:"SSH_EMAIL,required"`
	GrafanaURL     string `env:"GRAFANA_URL,required"`
	GrafanaSaToken string `env:"GRAFANA_SA_TOKEN,required"`

	RepoSavePath   string         `env:"REPO_SAVE_PATH,default=grafana-dashboards"`
	BaseBranch     string         `env:"BASE_BRANCH,default=main"`
	BranchPrefix   string         `env:"BRANCH_PREFIX,default=grafana-db-exporter-"`
	RepoClonePath  string         `env:"REPO_CLONE_PATH,default=./repo/"`
	RunMode        RunMode        `env:"RUN_MODE,default=one-time"`
	BranchStrategy BranchStrategy `env:"BRANCH_STRATEGY,default=new-branch"`
	SyncInterval   time.Duration  `env:"SYNC_INTERVAL,default=5m"`
	BranchTTL      time.Duration  `env:"BRANCH_TTL,default=24h"`

	SshKeyPassword        string `env:"SSH_KEY_PASSWORD"`
	SshKnownHostsPath     string `env:"SSH_KNOWN_HOSTS_PATH"`
	SshAcceptUnknownHosts bool   `env:"SSH_ACCEPT_UNKNOWN_HOSTS,default=false"`

	DeleteMissing         bool `env:"DELETE_MISSING,default=true"`
	EnableRetries         bool `env:"ENABLE_RETRIES,default=true"`
	NumOfRetries          uint `env:"NUM_OF_RETRIES,default=3"`
	RetriesBackoff        uint `env:"RETRIES_BACKOFF,default=5"`
	AddMissingNewlines    bool `env:"ADD_MISSING_NEWLINES,default=true"`
	DryRun                bool `env:"DRY_RUN,default=false"`
	IgnoreFolderStructure bool `env:"IGNORE_FOLDER_STRUCTURE,default=false"`
}

func Load() (*Config, error) {
	logger.Log.Debug().Msg("Loading configuration")
	cfg := &Config{}
	if err := parseEnv(cfg); err != nil {
		return nil, fmt.Errorf("environment parsing failed: %w", err)
	}

	cfg.RepoSavePath = filepath.Join(cfg.RepoClonePath, cfg.RepoSavePath)
	logger.Log.Debug().Str("FullRepoSavePath", cfg.RepoSavePath).Msg("Resolved save path")

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	logger.Log.Debug().
		Bool("DryRun", cfg.DryRun).
		Str("RunMode", string(cfg.RunMode)).
		Str("BranchStrategy", string(cfg.BranchStrategy)).
		Str("RepoSavePath", cfg.RepoSavePath).
		Msg("Configuration loaded")

	return cfg, nil
}

func (c *Config) Validate() error {
	logger.Log.Debug().Msg("Validating configuration")

	if !c.RunMode.IsValid() {
		return fmt.Errorf("invalid run mode: %s", c.RunMode)
	}

	if !c.BranchStrategy.IsValid() {
		return fmt.Errorf("invalid branch strategy: %s", c.BranchStrategy)
	}

	if c.RunMode == Periodic && c.SyncInterval < time.Second {
		return fmt.Errorf("sync interval must be at least 1 second, got %v", c.SyncInterval)
	}

	if c.BranchStrategy == ReuseBranch && c.BranchTTL < time.Minute {
		return fmt.Errorf("branch TTL must be at least 1 minute, got %v", c.BranchTTL)
	}

	logger.Log.Debug().Msg("Validating Grafana URL")
	if _, err := url.ParseRequestURI(c.GrafanaURL); err != nil {
		return fmt.Errorf("invalid Grafana URL: %w", err)
	}

	logger.Log.Debug().Str("SSHKeyPath", c.SSHKey).Msg("Checking SSH key file")
	if _, err := os.Stat(c.SSHKey); os.IsNotExist(err) {
		return fmt.Errorf("SSH key file not found: %s", c.SSHKey)
	}

	logger.Log.Debug().
		Bool("SshAcceptUnknownHosts", c.SshAcceptUnknownHosts).
		Str("SshKnownHostsPath", c.SshKnownHostsPath).
		Msg("Checking SSH known hosts configuration")
	if !c.SshAcceptUnknownHosts && c.SshKnownHostsPath != "" {
		if _, err := os.Stat(c.SshKnownHostsPath); os.IsNotExist(err) {
			return fmt.Errorf("SSH known hosts file not found: %s", c.SshKnownHostsPath)
		}
	}

	return nil
}

func parseEnv(cfg *Config) error {
	t := reflect.TypeOf(*cfg)
	v := reflect.ValueOf(cfg).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		tag := field.Tag.Get("env")
		if tag == "" {
			continue
		}

		parts := strings.Split(tag, ",")
		envName := parts[0]
		required := false
		defaultValue := ""

		for _, part := range parts[1:] {
			if part == "required" {
				required = true
			} else if strings.HasPrefix(part, "default=") {
				defaultValue = strings.TrimPrefix(part, "default=")
			}
		}

		envValue := os.Getenv(envName)
		if envValue == "" {
			if required {
				return fmt.Errorf("required environment variable %s not set", envName)
			}
			envValue = defaultValue
			logger.Log.Debug().Str("EnvVar", envName).Str("DefaultValue", defaultValue).Msg("Using default value")
		}

		if err := setField(value, envValue); err != nil {
			return fmt.Errorf("failed to set field %s: %w", field.Name, err)
		}
		logger.Log.Debug().Str("Field", field.Name).Msg("Field set successfully")
	}

	return nil
}

func setField(value reflect.Value, envValue string) error {
	switch value.Type() {
	case reflect.TypeOf(""):
		value.SetString(envValue)
	case reflect.TypeOf(true):
		b, err := strconv.ParseBool(envValue)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %s", envValue)
		}
		value.SetBool(b)
	case reflect.TypeOf(uint(0)):
		u, err := strconv.ParseUint(envValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value: %s", envValue)
		}
		value.SetUint(u)
	case reflect.TypeOf(RunMode("")):
		value.SetString(string(RunMode(envValue)))
	case reflect.TypeOf(BranchStrategy("")):
		value.SetString(string(BranchStrategy(envValue)))
	case reflect.TypeOf(time.Duration(0)):
		d, err := time.ParseDuration(envValue)
		if err != nil {
			return fmt.Errorf("invalid duration value: %s", envValue)
		}
		value.SetInt(int64(d))
	default:
		return fmt.Errorf("unsupported field type: %s", value.Type())
	}
	return nil
}
