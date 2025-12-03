package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config represents the complete application configuration
type Config struct {
	Jira     JiraConfig `koanf:"jira"`
	SLARules []SLARule  `koanf:"sla_rules"`
}

// JiraConfig holds Jira connection settings
type JiraConfig struct {
	BaseURL     string   `koanf:"base_url"`
	Email       string   `koanf:"email"`
	APIToken    string   `koanf:"api_token"`
	ProjectKeys []string `koanf:"project_keys"` // Support multiple projects
	ProjectKey  string   `koanf:"project_key"`  // Deprecated: kept for backward compatibility
}

// SLARule defines a threshold for bug age based on priority and status
type SLARule struct {
	Name       string   `koanf:"name"`
	Priority   string   `koanf:"priority"`
	Status     []string `koanf:"status"`
	MaxAgeDays float64  `koanf:"max_age_days"`
	Bucket     string   `koanf:"bucket"`
	Severity   int      `koanf:"severity"`
}

// Load reads configuration from a YAML file and environment variables
func Load(configPath string) (*Config, error) {
	k := koanf.New(".")

	// Load from config file
	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	// Load environment variables with JIRA_ prefix
	if err := k.Load(env.Provider("JIRA_", ".", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, "JIRA_"))
	}), nil); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	// Unmarshal into Config struct
	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Interpolate environment variables in config values
	if err := interpolateEnvVars(&cfg); err != nil {
		return nil, fmt.Errorf("failed to interpolate environment variables: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// interpolateEnvVars replaces ${VAR} patterns with environment variable values
func interpolateEnvVars(cfg *Config) error {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)

	// Interpolate API token
	if matches := re.FindStringSubmatch(cfg.Jira.APIToken); len(matches) > 1 {
		envVar := matches[1]
		value := os.Getenv(envVar)
		if value == "" {
			return fmt.Errorf("environment variable %s is not set", envVar)
		}
		cfg.Jira.APIToken = value
	}

	return nil
}

// Validate checks that all required configuration fields are present and valid
func (c *Config) Validate() error {
	// Validate Jira config
	if c.Jira.BaseURL == "" {
		return fmt.Errorf("jira.base_url is required")
	}
	if c.Jira.Email == "" {
		return fmt.Errorf("jira.email is required")
	}
	if c.Jira.APIToken == "" {
		return fmt.Errorf("jira.api_token is required")
	}
	// Support both project_key (single, deprecated) and project_keys (multiple)
	if len(c.Jira.ProjectKeys) == 0 && c.Jira.ProjectKey == "" {
		return fmt.Errorf("either jira.project_keys or jira.project_key is required")
	}

	// If old project_key is used, migrate it to project_keys
	if c.Jira.ProjectKey != "" && len(c.Jira.ProjectKeys) == 0 {
		c.Jira.ProjectKeys = []string{c.Jira.ProjectKey}
	}

	// Validate SLA rules
	if len(c.SLARules) == 0 {
		return fmt.Errorf("at least one SLA rule is required")
	}

	for i, rule := range c.SLARules {
		if rule.Name == "" {
			return fmt.Errorf("sla_rules[%d].name is required", i)
		}
		if rule.MaxAgeDays < 0 {
			return fmt.Errorf("sla_rules[%d].max_age_days must be non-negative", i)
		}
		if rule.Bucket == "" {
			return fmt.Errorf("sla_rules[%d].bucket is required", i)
		}
		if rule.Severity < 1 {
			return fmt.Errorf("sla_rules[%d].severity must be >= 1", i)
		}
	}

	return nil
}
