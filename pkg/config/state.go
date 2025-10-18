package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	// File and directory permissions for config
	permConfigDir = 0755 // Owner rwx, others rx (rwxr-xr-x)
	permStateFile = 0600 // Owner read/write only (rw-------)
)

// StateChange represents a change in instance state for cost tracking
type StateChange struct {
	State     string    `json:"state"`      // "running", "stopped", "terminated"
	Timestamp time.Time `json:"timestamp"`
}

// Instance represents a tracked EC2 instance with its metadata
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
	AMIBase       string    `json:"ami_base,omitempty"`
	S3Bucket      string    `json:"s3_bucket,omitempty"`     // S3 bucket for data sync
	S3MountPath   string    `json:"s3_mount_path,omitempty"` // Local path where S3 is mounted
	EBSSize       int       `json:"ebs_size,omitempty"`      // EBS volume size in GB
	StateChanges  []StateChange `json:"state_changes,omitempty"` // History of state changes for cost tracking
}

// LocalState manages the local state file tracking active instances
type LocalState struct {
	Instances map[string]*Instance `json:"instances"`
	KeyPairs  map[string]string    `json:"key_pairs"` // name -> private key path
}

// GetConfigDir returns the path to the aws-jupyter configuration directory
func GetConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws-jupyter")
}

// EnsureConfigDir creates the configuration directory if it doesn't exist
func EnsureConfigDir() error {
	configDir := GetConfigDir()
	return os.MkdirAll(filepath.Join(configDir, "environments"), permConfigDir)
}

// LoadState loads the local state file or creates a new one if it doesn't exist
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

// Save writes the current state to the local state file
func (s *LocalState) Save() error {
	statePath := filepath.Join(GetConfigDir(), "state.json")
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath, data, permStateFile)
}

// RecordStateChange records a state change for an instance
func (i *Instance) RecordStateChange(state string) {
	// Don't record duplicate state changes
	if len(i.StateChanges) > 0 {
		lastState := i.StateChanges[len(i.StateChanges)-1].State
		if lastState == state {
			return
		}
	}

	i.StateChanges = append(i.StateChanges, StateChange{
		State:     state,
		Timestamp: time.Now(),
	})
}
