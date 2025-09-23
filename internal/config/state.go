package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Instance struct {
	ID            string    `json:"id"`
	Environment   string    `json:"environment"`
	InstanceType  string    `json:"instance_type"`
	PublicIP      string    `json:"public_ip"`
	KeyPair       string    `json:"key_pair"`
	LaunchedAt    time.Time `json:"launched_at"`
	IdleTimeout   string    `json:"idle_timeout"`
	TunnelPID     int       `json:"tunnel_pid,omitempty"`
	Region        string    `json:"region"`
	SecurityGroup string    `json:"security_group"`
}

type LocalState struct {
	Instances map[string]*Instance `json:"instances"`
	KeyPairs  map[string]string    `json:"key_pairs"` // name -> private key path
}

func GetConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws-jupyter")
}

func EnsureConfigDir() error {
	configDir := GetConfigDir()
	return os.MkdirAll(filepath.Join(configDir, "environments"), 0755)
}

func LoadState() (*LocalState, error) {
	statePath := filepath.Join(GetConfigDir(), "state.json")

	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return &LocalState{
			Instances: make(map[string]*Instance),
			KeyPairs:  make(map[string]string),
		}, nil
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, err
	}

	var state LocalState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	if state.Instances == nil {
		state.Instances = make(map[string]*Instance)
	}
	if state.KeyPairs == nil {
		state.KeyPairs = make(map[string]string)
	}

	return &state, nil
}

func (s *LocalState) Save() error {
	statePath := filepath.Join(GetConfigDir(), "state.json")
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath, data, 0600)
}
