package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Environment struct {
	Name              string            `yaml:"name"`
	InstanceType      string            `yaml:"instance_type"`
	AMIBase           string            `yaml:"ami_base"`
	EBSVolumeSize     int               `yaml:"ebs_volume_size"`
	Packages          []string          `yaml:"packages"`
	PipPackages       []string          `yaml:"pip_packages"`
	RPackages         []string          `yaml:"r_packages,omitempty"`
	JupyterExtensions []string          `yaml:"jupyter_extensions"`
	EnvironmentVars   map[string]string `yaml:"environment_vars"`
}

func LoadEnvironment(name string) (*Environment, error) {
	// Try user config first, then built-in
	userPath := filepath.Join(GetConfigDir(), "environments", name+".yaml")
	builtinPath := filepath.Join("environments", name+".yaml")

	var path string
	if _, err := os.Stat(userPath); err == nil {
		path = userPath
	} else if _, err := os.Stat(builtinPath); err == nil {
		path = builtinPath
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

func ListEnvironments() ([]string, error) {
	var envs []string

	// Check built-in environments
	if entries, err := os.ReadDir("environments"); err == nil {
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) == ".yaml" {
				envs = append(envs, entry.Name()[:len(entry.Name())-5])
			}
		}
	}

	// Check user environments
	userEnvDir := filepath.Join(GetConfigDir(), "environments")
	if entries, err := os.ReadDir(userEnvDir); err == nil {
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) == ".yaml" {
				name := entry.Name()[:len(entry.Name())-5]
				// Avoid duplicates
				found := false
				for _, existing := range envs {
					if existing == name {
						found = true
						break
					}
				}
				if !found {
					envs = append(envs, name)
				}
			}
		}
	}

	return envs, nil
}
