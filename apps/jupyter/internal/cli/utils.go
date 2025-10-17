package cli

import (
	"fmt"
	"os"
	"syscall"
)

// killProcess kills a process by PID
func killProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send SIGTERM for graceful shutdown
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// If SIGTERM fails, try SIGKILL
		if killErr := process.Kill(); killErr != nil {
			return fmt.Errorf("failed to kill process: %w", killErr)
		}
	}

	return nil
}
