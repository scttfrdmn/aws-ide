package config

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/scttfrdmn/lens/pkg/dcv"
)

// GenerateUserData creates a cloud-init user data script for QGIS with DCV
func GenerateUserData(env *QGISEnvironment, idleTimeoutSeconds int) (string, error) {
	script := generateUserDataScript(env, idleTimeoutSeconds)
	// AWS expects user data to be base64 encoded
	encoded := base64.StdEncoding.EncodeToString([]byte(script))
	return encoded, nil
}

// generateUserDataScript creates the actual bash script for QGIS + DCV setup
func generateUserDataScript(env *QGISEnvironment, idleTimeoutSeconds int) string {
	var sb strings.Builder

	// Start with bash shebang and error handling
	sb.WriteString("#!/bin/bash\n")
	sb.WriteString("set -e\n")
	sb.WriteString("set -x\n\n")

	// Log file for debugging
	sb.WriteString("exec > >(tee /var/log/user-data.log)\n")
	sb.WriteString("exec 2>&1\n\n")

	// Create progress log file
	sb.WriteString("# Setup progress tracking\n")
	sb.WriteString("PROGRESS_LOG=\"/var/log/setup-progress.log\"\n")
	sb.WriteString("touch $PROGRESS_LOG\n")
	sb.WriteString("chmod 644 $PROGRESS_LOG\n\n")

	sb.WriteString("log_progress() {\n")
	sb.WriteString("  echo \"STEP:$1\" | tee -a $PROGRESS_LOG\n")
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("log_progress 'Starting lens-qgis environment setup (%s)'\n", env.Name))
	sb.WriteString(fmt.Sprintf("echo 'Environment: %s'\n", env.Description))
	sb.WriteString(fmt.Sprintf("echo 'GPU Required: %v'\n\n", env.RequiresGPU))

	// Update system
	sb.WriteString("# Update system packages\n")
	sb.WriteString("log_progress 'Updating system packages'\n")
	sb.WriteString("apt-get update -y\n")
	sb.WriteString("DEBIAN_FRONTEND=noninteractive apt-get upgrade -y\n\n")

	// Install SSM Agent
	sb.WriteString("# Install AWS Systems Manager Agent\n")
	sb.WriteString("log_progress 'Installing SSM Agent'\n")
	sb.WriteString("snap install amazon-ssm-agent --classic\n")
	sb.WriteString("systemctl enable snap.amazon-ssm-agent.amazon-ssm-agent.service\n")
	sb.WriteString("systemctl start snap.amazon-ssm-agent.amazon-ssm-agent.service\n\n")

	// GPU setup if required
	if env.RequiresGPU {
		sb.WriteString(dcv.GenerateGPUSetupScript())
	}

	// Install desktop environment (MATE for lightweight performance)
	sb.WriteString(dcv.GenerateDesktopInstallScript("mate"))

	// Install QGIS and related packages
	sb.WriteString(generateQGISInstallScript(env))

	// Install NICE DCV Server
	sb.WriteString(dcv.GenerateDCVInstallScript(env.DCVConfig))

	// Configure DCV session
	sb.WriteString(dcv.GenerateDCVSessionScript(env.DCVConfig))

	// Configure QGIS to auto-launch on desktop startup
	sb.WriteString(generateQGISAutoLaunchScript())

	// Setup idle timeout monitoring
	if idleTimeoutSeconds > 0 {
		sb.WriteString(dcv.GenerateIdleMonitorScript(env.DCVConfig, idleTimeoutSeconds))
	}

	// Final setup steps
	sb.WriteString("# Finalize setup\n")
	sb.WriteString("log_progress 'QGIS + DCV Desktop setup complete'\n")
	sb.WriteString("echo 'SETUP_COMPLETE' > /var/log/qgis-ready\n\n")

	// Reboot to ensure all changes take effect
	sb.WriteString("# Reboot to apply all changes\n")
	sb.WriteString("log_progress 'Rebooting system'\n")
	sb.WriteString("reboot\n")

	return sb.String()
}

// generateQGISInstallScript creates the QGIS installation script
func generateQGISInstallScript(env *QGISEnvironment) string {
	var sb strings.Builder

	sb.WriteString("# Install QGIS and related packages\n")
	sb.WriteString("log_progress 'Installing QGIS'\n\n")

	// Add QGIS repository
	sb.WriteString("# Add QGIS repository\n")
	sb.WriteString("wget -qO - https://qgis.org/downloads/qgis-2022.gpg.key | gpg --no-default-keyring --keyring gnupg-ring:/etc/apt/trusted.gpg.d/qgis-archive.gpg --import\n")
	sb.WriteString("chmod a+r /etc/apt/trusted.gpg.d/qgis-archive.gpg\n")
	sb.WriteString("echo 'deb https://qgis.org/ubuntu jammy main' > /etc/apt/sources.list.d/qgis.list\n")
	sb.WriteString("apt-get update\n\n")

	// Install QGIS and packages
	if len(env.Packages) > 0 {
		sb.WriteString("# Install QGIS and related packages\n")
		sb.WriteString("DEBIAN_FRONTEND=noninteractive apt-get install -y \\\n")
		for i, pkg := range env.Packages {
			if i == len(env.Packages)-1 {
				sb.WriteString("  " + pkg + "\n\n")
			} else {
				sb.WriteString("  " + pkg + " \\\n")
			}
		}
	}

	// Install QGIS plugins if specified
	if len(env.QGISPlugins) > 0 {
		sb.WriteString("# Configure QGIS plugins\n")
		sb.WriteString("# Plugins will be available in QGIS Plugin Manager:\n")
		for _, plugin := range env.QGISPlugins {
			sb.WriteString(fmt.Sprintf("# - %s\n", plugin))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// generateQGISAutoLaunchScript creates a script to auto-launch QGIS on desktop startup
func generateQGISAutoLaunchScript() string {
	var sb strings.Builder

	sb.WriteString("# Configure QGIS to auto-launch on desktop startup\n")
	sb.WriteString("log_progress 'Configuring QGIS auto-launch'\n\n")

	sb.WriteString("# Create autostart directory for ubuntu user\n")
	sb.WriteString("mkdir -p /home/ubuntu/.config/autostart\n\n")

	sb.WriteString("# Create QGIS desktop entry for autostart\n")
	sb.WriteString("cat > /home/ubuntu/.config/autostart/qgis.desktop << 'QGISAUTOSTART'\n")
	sb.WriteString("[Desktop Entry]\n")
	sb.WriteString("Type=Application\n")
	sb.WriteString("Name=QGIS Desktop\n")
	sb.WriteString("Exec=qgis\n")
	sb.WriteString("Terminal=false\n")
	sb.WriteString("StartupNotify=true\n")
	sb.WriteString("QGISAUTOSTART\n\n")

	sb.WriteString("# Set proper ownership\n")
	sb.WriteString("chown -R ubuntu:ubuntu /home/ubuntu/.config\n\n")

	sb.WriteString("# Create desktop shortcut\n")
	sb.WriteString("mkdir -p /home/ubuntu/Desktop\n")
	sb.WriteString("cp /usr/share/applications/org.qgis.qgis.desktop /home/ubuntu/Desktop/qgis.desktop 2>/dev/null || true\n")
	sb.WriteString("chmod +x /home/ubuntu/Desktop/qgis.desktop 2>/dev/null || true\n")
	sb.WriteString("chown ubuntu:ubuntu /home/ubuntu/Desktop/qgis.desktop 2>/dev/null || true\n\n")

	return sb.String()
}
