package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"grafana-db-exporter/internal/logger"
)

type Config struct {
	SSHURL         string `env:"SSH_URL,required"`
	SSHKey         string `env:"SSH_KEY,required"`
	SSHUser        string `env:"SSH_USER,required"`
	SSHEmail       string `env:"SSH_EMAIL,required"`
	RepoSavePath   string `env:"REPO_SAVE_PATH,required"`
	GrafanaURL     string `env:"GRAFANA_URL,required"`
	GrafanaSaToken string `env:"GRAFANA_SA_TOKEN,required"`

	BaseBranch            string `env:"BASE_BRANCH,default=main"`
	BranchPrefix          string `env:"BRANCH_PREFIX,default=grafana-db-exporter-"`
	SshKeyPassword        string `env:"SSH_KEY_PASSWORD"`
	SshAcceptUnknownHosts bool   `env:"SSH_ACCEPT_UNKNOWN_HOSTS,default=false"`
	SshKnownHostsPath     string `env:"SSH_KNOWN_HOSTS_PATH"`

	RepoClonePath string `env:"REPO_CLONE_PATH,default=./repo/"`
	DeleteMissing bool   `env:"DELETE_MISSING,default=true"`

	EnableRetries  bool `env:"ENABLE_RETRIES,default=true"`
	NumOfRetries   uint `env:"NUM_OF_RETRIES,default=3"`
	RetriesBackoff uint `env:"RETRIES_BACKOFF,default=5"`

	AddMissingNewlines bool `env:"ADD_MISSING_NEWLINES,default=true"`
	DryRun             bool `env:"DRY_RUN,default=false"`
}

func Load() (*Config, error) {
	logger.Log.Debug().Msg("Starting configuration loading process")
	cfg := &Config{}
	if err := parseEnv(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	cfg.RepoSavePath = filepath.Join(cfg.RepoClonePath, cfg.RepoSavePath)
	logger.Log.Debug().Str("FullRepoSavePath", cfg.RepoSavePath).Msg("Full RepoSavePath")

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	logger.Log.Debug().Msg("Configuration validation completed successfully")

	logger.Log.Debug().Bool("DryRun", cfg.DryRun).Msg("Dry run mode")

	logger.Log.Debug().Msg("Configuration loading process completed")
	return cfg, nil
}

func (c *Config) Validate() error {
	logger.Log.Debug().Msg("Validating Grafana URL")
	if _, err := url.ParseRequestURI(c.GrafanaURL); err != nil {
		return fmt.Errorf("invalid Grafana URL: %w", err)
	}

	logger.Log.Debug().Str("SSHKeyPath", c.SSHKey).Msg("Checking SSH key file")
	if _, err := os.Stat(c.SSHKey); os.IsNotExist(err) {
		return fmt.Errorf("SSH key file does not exist: %s", c.SSHKey)
	}

	logger.Log.Debug().
		Bool("SshAcceptUnknownHosts", c.SshAcceptUnknownHosts).
		Str("SshKnownHostsPath", c.SshKnownHostsPath).
		Msg("Checking SSH known hosts configuration")
	if !c.SshAcceptUnknownHosts && c.SshKnownHostsPath != "" {
		if _, err := os.Stat(c.SshKnownHostsPath); os.IsNotExist(err) {
			return fmt.Errorf("SSH known hosts file does not exist: %s", c.SshKnownHostsPath)
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
				return fmt.Errorf("required environment variable %s is not set", envName)
			}
			if defaultValue != "" {
				envValue = defaultValue
				logger.Log.Warn().Str("EnvVar", envName).Str("DefaultValue", defaultValue).Msg("Unset env var, using default value")
			}
		}

		if err := setField(value, envValue); err != nil {
			return fmt.Errorf("failed to set field %s: %w", field.Name, err)
		}
		logger.Log.Debug().Str("Field", field.Name).Msg("Field set successfully")
	}

	return nil
}

func setField(value reflect.Value, envValue string) error {
	switch value.Kind() {
	case reflect.String:
		value.SetString(envValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(envValue)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %s", envValue)
		}
		value.SetBool(boolValue)
	case reflect.Uint:
		uintValue, err := strconv.ParseUint(envValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value: %s", envValue)
		}
		value.SetUint(uintValue)
	default:
		return fmt.Errorf("unsupported field type: %s", value.Type())
	}
	return nil
}
