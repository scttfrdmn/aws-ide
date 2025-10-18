package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ExportPath defines a file or directory path to export from an instance
type ExportPath struct {
	Path            string   `yaml:"path"`
	Description     string   `yaml:"description"`
	Optional        bool     `yaml:"optional,omitempty"`
	GenerateCommand string   `yaml:"generate_command,omitempty"`
	Exclude         []string `yaml:"exclude,omitempty"`
}

// RestoreCommand defines a command to run after importing configuration
type RestoreCommand struct {
	Description string `yaml:"description"`
	Command     string `yaml:"command"`
	Timeout     int    `yaml:"timeout"`
	Optional    bool   `yaml:"optional,omitempty"`
}

// ExportConfig defines what configuration to export/import for an application
type ExportConfig struct {
	Name            string           `yaml:"name"`
	Description     string           `yaml:"description"`
	App             string           `yaml:"app"`
	ExportPaths     []ExportPath     `yaml:"export_paths"`
	RestoreCommands []RestoreCommand `yaml:"restore_commands"`
}

// LoadExportConfig loads an export configuration by name from built-in or user configs
func LoadExportConfig(name string) (*ExportConfig, error) {
	// Try user config first, then built-in, then Homebrew pkgshare
	userPath := filepath.Join(GetConfigDir(), "configs", name+".yaml")
	builtinPath := filepath.Join("configs", name+".yaml")
	homebrewPath := filepath.Join("/opt/homebrew/share/aws-ide/configs", name+".yaml")
	linuxbrewPath := filepath.Join("/home/linuxbrew/.linuxbrew/share/aws-ide/configs", name+".yaml")

	var path string
	if _, err := os.Stat(userPath); err == nil {
		path = userPath
	} else if _, err := os.Stat(builtinPath); err == nil {
		path = builtinPath
	} else if _, err := os.Stat(homebrewPath); err == nil {
		path = homebrewPath
	} else if _, err := os.Stat(linuxbrewPath); err == nil {
		path = linuxbrewPath
	} else {
		return nil, fmt.Errorf("export config %s not found", name)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ExportConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// ListExportConfigs returns a list of all available export config names from built-in and user configs
func ListExportConfigs() ([]string, error) {
	var configs []string
	seen := make(map[string]bool)

	// Check directories in order of precedence
	dirs := []string{
		filepath.Join(GetConfigDir(), "configs"),
		"configs",
		"/opt/homebrew/share/aws-ide/configs",
		"/home/linuxbrew/.linuxbrew/share/aws-ide/configs",
	}

	for _, dir := range dirs {
		if entries, err := os.ReadDir(dir); err == nil {
			for _, entry := range entries {
				if filepath.Ext(entry.Name()) == ".yaml" {
					name := entry.Name()[:len(entry.Name())-5]
					if !seen[name] {
						configs = append(configs, name)
						seen[name] = true
					}
				}
			}
		}
	}

	return configs, nil
}

// GetDefaultConfigForApp returns the default export config name for a given app
func GetDefaultConfigForApp(app string) string {
	return fmt.Sprintf("%s-default", app)
}
