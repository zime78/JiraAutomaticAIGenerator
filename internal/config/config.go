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
	CLIPath      string
	ChannelPaths [3]string // 채널별 프로젝트 경로
	Enabled      bool
	Model        string // Claude 모델 (claude-sonnet-4-20250514, claude-opus-4-20250514 등)
}

// Available Claude models
var AvailableModels = []string{
	"claude-sonnet-4-20250514",
	"claude-opus-4-20250514",
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
	config.Claude.ChannelPaths = [3]string{
		claudeSection.Key("project_path_1").MustString(""),
		claudeSection.Key("project_path_2").MustString(""),
		claudeSection.Key("project_path_3").MustString(""),
	}
	config.Claude.Enabled = claudeSection.Key("enabled").MustBool(false)
	config.Claude.Model = claudeSection.Key("model").MustString("claude-sonnet-4-20250514")

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

// Save writes configuration to the specified INI file
func (c *Config) Save(path string) error {
	cfg := ini.Empty()

	// Jira section
	jiraSection, _ := cfg.NewSection("jira")
	jiraSection.NewKey("url", c.Jira.URL)
	jiraSection.NewKey("email", c.Jira.Email)
	jiraSection.NewKey("api_key", c.Jira.APIKey)

	// Output section
	outputSection, _ := cfg.NewSection("output")
	outputSection.NewKey("dir", c.Output.Dir)

	// AI section
	aiSection, _ := cfg.NewSection("ai")
	aiSection.NewKey("prompt_template", c.AI.PromptTemplate)

	// Claude section
	claudeSection, _ := cfg.NewSection("claude")
	claudeSection.NewKey("cli_path", c.Claude.CLIPath)
	claudeSection.NewKey("project_path_1", c.Claude.ChannelPaths[0])
	claudeSection.NewKey("project_path_2", c.Claude.ChannelPaths[1])
	claudeSection.NewKey("project_path_3", c.Claude.ChannelPaths[2])
	claudeSection.NewKey("enabled", fmt.Sprintf("%v", c.Claude.Enabled))
	claudeSection.NewKey("model", c.Claude.Model)

	return cfg.SaveTo(path)
}

// SaveDefault saves configuration to the default location
func (c *Config) SaveDefault() error {
	// Try current directory first
	if _, err := os.Stat("config.ini"); err == nil {
		return c.Save("config.ini")
	}

	// Try user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".jira-ai-generator")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.ini")
	return c.Save(configPath)
}

// GetConfigPath returns the path where config would be saved
func GetConfigPath() string {
	// Try current directory first
	if _, err := os.Stat("config.ini"); err == nil {
		return "config.ini"
	}

	// Return user home directory path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "config.ini"
	}
	return filepath.Join(homeDir, ".jira-ai-generator", "config.ini")
}
