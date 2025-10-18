package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// UserConfig represents user preferences and defaults
type UserConfig struct {
	// AWS settings
	DefaultRegion  string `yaml:"default_region,omitempty"`
	DefaultProfile string `yaml:"default_profile,omitempty"`

	// Instance defaults
	DefaultInstanceType string `yaml:"default_instance_type,omitempty"`
	DefaultEBSSize      int    `yaml:"default_ebs_size,omitempty"`
	DefaultAMIBase      string `yaml:"default_ami_base,omitempty"`

	// Networking defaults
	DefaultSubnetType string `yaml:"default_subnet_type,omitempty"` // "public" or "private"
	PreferIPv6        bool   `yaml:"prefer_ipv6,omitempty"`

	// Behavior settings
	IdleTimeout        string `yaml:"idle_timeout,omitempty"`
	AutoTerminate      bool   `yaml:"auto_terminate,omitempty"`
	ConfirmDestructive bool   `yaml:"confirm_destructive,omitempty"` // Confirm before terminate/delete

	// Cost tracking
	EnableCostTracking bool    `yaml:"enable_cost_tracking,omitempty"`
	CostAlertThreshold float64 `yaml:"cost_alert_threshold,omitempty"` // Alert when monthly cost exceeds this

	// App-specific settings
	Jupyter *AppConfig `yaml:"jupyter,omitempty"`
	RStudio *AppConfig `yaml:"rstudio,omitempty"`
	VSCode  *AppConfig `yaml:"vscode,omitempty"`
}

// AppConfig contains app-specific configuration
type AppConfig struct {
	DefaultEnvironment  string `yaml:"default_environment,omitempty"`
	DefaultInstanceType string `yaml:"default_instance_type,omitempty"`
	DefaultEBSSize      int    `yaml:"default_ebs_size,omitempty"`
	Port                int    `yaml:"port,omitempty"`
}

// GetUserConfigPath returns the path to the user config file
func GetUserConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws-ide", "config.yaml")
}

// LoadUserConfig loads the user configuration file
// Returns default config if file doesn't exist
func LoadUserConfig() (*UserConfig, error) {
	configPath := GetUserConfigPath()

	// Return defaults if file doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return getDefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config UserConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Fill in defaults for any missing values
	applyDefaults(&config)

	return &config, nil
}

// SaveUserConfig saves the user configuration to file
func SaveUserConfig(config *UserConfig) error {
	configPath := GetUserConfigPath()

	// Ensure directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, permConfigDir); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Add header comment
	header := `# aws-ide User Configuration
# This file contains default settings for all aws-ide tools
# Edit this file to customize your preferences

`
	fullContent := header + string(data)

	if err := os.WriteFile(configPath, []byte(fullContent), permStateFile); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// InitUserConfig creates a new config file with defaults
func InitUserConfig() error {
	config := getDefaultConfig()
	return SaveUserConfig(config)
}

// getDefaultConfig returns a config with sensible defaults
func getDefaultConfig() *UserConfig {
	return &UserConfig{
		DefaultRegion:       "", // Use AWS SDK default
		DefaultProfile:      "", // Use AWS SDK default
		DefaultInstanceType: "t4g.medium",
		DefaultEBSSize:      20,
		DefaultAMIBase:      "ubuntu-22.04-arm64",
		DefaultSubnetType:   "public",
		PreferIPv6:          false,
		IdleTimeout:         "4h",
		AutoTerminate:       false,
		ConfirmDestructive:  true,
		EnableCostTracking:  true,
		CostAlertThreshold:  100.0, // $100/month
		Jupyter: &AppConfig{
			DefaultEnvironment:  "data-science",
			DefaultInstanceType: "", // Use global default
			Port:                8888,
		},
		RStudio: &AppConfig{
			DefaultEnvironment:  "r-statistics",
			DefaultInstanceType: "", // Use global default
			Port:                8787,
		},
		VSCode: &AppConfig{
			DefaultEnvironment:  "web",
			DefaultInstanceType: "", // Use global default
			Port:                8080,
		},
	}
}

// applyDefaults fills in missing values with defaults
func applyDefaults(config *UserConfig) {
	defaults := getDefaultConfig()

	if config.DefaultInstanceType == "" {
		config.DefaultInstanceType = defaults.DefaultInstanceType
	}
	if config.DefaultEBSSize == 0 {
		config.DefaultEBSSize = defaults.DefaultEBSSize
	}
	if config.DefaultAMIBase == "" {
		config.DefaultAMIBase = defaults.DefaultAMIBase
	}
	if config.DefaultSubnetType == "" {
		config.DefaultSubnetType = defaults.DefaultSubnetType
	}
	if config.IdleTimeout == "" {
		config.IdleTimeout = defaults.IdleTimeout
	}
	if config.CostAlertThreshold == 0 {
		config.CostAlertThreshold = defaults.CostAlertThreshold
	}

	// App-specific defaults
	if config.Jupyter == nil {
		config.Jupyter = defaults.Jupyter
	} else if config.Jupyter.Port == 0 {
		config.Jupyter.Port = defaults.Jupyter.Port
	}

	if config.RStudio == nil {
		config.RStudio = defaults.RStudio
	} else if config.RStudio.Port == 0 {
		config.RStudio.Port = defaults.RStudio.Port
	}

	if config.VSCode == nil {
		config.VSCode = defaults.VSCode
	} else if config.VSCode.Port == 0 {
		config.VSCode.Port = defaults.VSCode.Port
	}
}

// GetAppConfig returns the app-specific config, or defaults if not set
func (c *UserConfig) GetAppConfig(appName string) *AppConfig {
	var appConfig *AppConfig

	switch appName {
	case "jupyter":
		appConfig = c.Jupyter
	case "rstudio":
		appConfig = c.RStudio
	case "vscode":
		appConfig = c.VSCode
	default:
		return &AppConfig{}
	}

	if appConfig == nil {
		return &AppConfig{}
	}

	return appConfig
}

// GetInstanceType returns the instance type to use, checking app config first, then global
func (c *UserConfig) GetInstanceType(appName string) string {
	appConfig := c.GetAppConfig(appName)
	if appConfig.DefaultInstanceType != "" {
		return appConfig.DefaultInstanceType
	}
	return c.DefaultInstanceType
}

// GetEBSSize returns the EBS size to use, checking app config first, then global
func (c *UserConfig) GetEBSSize(appName string) int {
	appConfig := c.GetAppConfig(appName)
	if appConfig.DefaultEBSSize > 0 {
		return appConfig.DefaultEBSSize
	}
	return c.DefaultEBSSize
}
