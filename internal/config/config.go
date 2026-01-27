package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

// Config holds all application configuration
type Config struct {
	Jira   JiraConfig
	Output OutputConfig
	AI     AIConfig
	Claude ClaudeConfig
}

// JiraConfig holds Jira-related settings
type JiraConfig struct {
	URL    string
	Email  string
	APIKey string
}

// OutputConfig holds output-related settings
type OutputConfig struct {
	Dir string
}

// AIConfig holds AI-related settings
type AIConfig struct {
	PromptTemplate string
}

// ClaudeConfig holds Claude Code CLI settings
type ClaudeConfig struct {
	CLIPath     string
	WorkDir     string
	ProjectPath string
	Enabled     bool
}

// Load reads configuration from the specified INI file
func Load(path string) (*Config, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	config := &Config{}

	// Jira section
	jiraSection := cfg.Section("jira")
	config.Jira.URL = jiraSection.Key("url").String()
	config.Jira.Email = jiraSection.Key("email").String()
	config.Jira.APIKey = jiraSection.Key("api_key").String()

	// Output section
	outputSection := cfg.Section("output")
	config.Output.Dir = outputSection.Key("dir").MustString("./output")

	// AI section
	aiSection := cfg.Section("ai")
	config.AI.PromptTemplate = aiSection.Key("prompt_template").String()

	// Claude section
	claudeSection := cfg.Section("claude")
	config.Claude.CLIPath = claudeSection.Key("cli_path").MustString("claude")
	config.Claude.WorkDir = claudeSection.Key("work_dir").MustString("./")
	config.Claude.ProjectPath = claudeSection.Key("project_path").MustString("")
	config.Claude.Enabled = claudeSection.Key("enabled").MustBool(false)

	return config, nil
}

// LoadDefault attempts to load config from default locations
func LoadDefault() (*Config, error) {
	// Try current directory first
	if _, err := os.Stat("config.ini"); err == nil {
		return Load("config.ini")
	}

	// Try user home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".jira-ai-generator", "config.ini")
		if _, err := os.Stat(configPath); err == nil {
			return Load(configPath)
		}
	}

	return nil, fmt.Errorf("config.ini not found")
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Jira.URL == "" {
		return fmt.Errorf("jira.url is required")
	}
	if c.Jira.Email == "" {
		return fmt.Errorf("jira.email is required")
	}
	if c.Jira.APIKey == "" {
		return fmt.Errorf("jira.api_key is required")
	}
	return nil
}
