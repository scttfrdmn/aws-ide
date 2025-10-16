package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Environment defines a complete configuration for a Jupyter instance including packages and settings
type Environment struct {
	Name              string            `yaml:"name"`
	InstanceType      string            `yaml:"instance_type"`
	AMIBase           string            `yaml:"ami_base"`
	EBSVolumeSize     int               `yaml:"ebs_volume_size"`
	Packages          []string          `yaml:"packages"`
	PipPackages       []string          `yaml:"pip_packages"`
	RPackages         []string          `yaml:"r_packages,omitempty"`
	JuliaPackages     []string          `yaml:"julia_packages,omitempty"`
	JupyterExtensions []string          `yaml:"jupyter_extensions"`
	EnvironmentVars   map[string]string `yaml:"environment_vars"`
}

// LoadEnvironment loads an environment configuration by name from built-in or user configs
func LoadEnvironment(name string) (*Environment, error) {
	// Try user config first, then built-in, then Homebrew pkgshare
	userPath := filepath.Join(GetConfigDir(), "environments", name+".yaml")
	builtinPath := filepath.Join("environments", name+".yaml")
	homebrewPath := filepath.Join("/opt/homebrew/share/aws-jupyter/environments", name+".yaml")
	linuxbrewPath := filepath.Join("/home/linuxbrew/.linuxbrew/share/aws-jupyter/environments", name+".yaml")

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
		return nil, fmt.Errorf("environment %s not found", name)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var env Environment
	if err := yaml.Unmarshal(data, &env); err != nil {
		return nil, err
	}

	return &env, nil
}

// ListEnvironments returns a list of all available environment names from built-in and user configs
func ListEnvironments() ([]string, error) {
	var envs []string
	seen := make(map[string]bool)

	// Check directories in order of precedence
	dirs := []string{
		filepath.Join(GetConfigDir(), "environments"),
		"environments",
		"/opt/homebrew/share/aws-jupyter/environments",
		"/home/linuxbrew/.linuxbrew/share/aws-jupyter/environments",
	}

	for _, dir := range dirs {
		if entries, err := os.ReadDir(dir); err == nil {
			for _, entry := range entries {
				if filepath.Ext(entry.Name()) == ".yaml" {
					name := entry.Name()[:len(entry.Name())-5]
					if !seen[name] {
						envs = append(envs, name)
						seen[name] = true
					}
				}
			}
		}
	}

	return envs, nil
}
