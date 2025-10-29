package config

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/scttfrdmn/lens/pkg/dcv"
)

// GenerateUserData creates a cloud-init user data script for QGIS with DCV
func GenerateUserData(env *QGISEnvironment, idleTimeoutSeconds int, skipSystemUpgrade bool) (string, error) {
	script := generateUserDataScript(env, idleTimeoutSeconds, skipSystemUpgrade)
	// AWS expects user data to be base64 encoded
	encoded := base64.StdEncoding.EncodeToString([]byte(script))
	return encoded, nil
}

// generateUserDataScript creates the actual bash script for QGIS + DCV setup
func generateUserDataScript(env *QGISEnvironment, idleTimeoutSeconds int, skipSystemUpgrade bool) string {
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

	// Configure APT for faster parallel downloads with multiple AWS mirrors
	sb.WriteString("# Configure APT for parallel downloads across multiple AWS EC2 mirrors\n")
	sb.WriteString("echo 'Acquire::Queue-Mode \"host\";' > /etc/apt/apt.conf.d/99parallel\n")
	sb.WriteString("echo 'Acquire::http::Pipeline-Depth \"5\";' >> /etc/apt/apt.conf.d/99parallel\n")
	sb.WriteString("echo 'APT::Acquire::Max-Parallel-Downloads \"16\";' >> /etc/apt/apt.conf.d/99parallel\n\n")

	// Add multiple AWS EC2 mirrors for faster parallel downloads
	sb.WriteString("# Add multiple AWS EC2 mirrors for load balancing\n")
	sb.WriteString("cat > /etc/apt/sources.list.d/aws-mirrors.list << 'EOF'\n")
	sb.WriteString("deb mirror+file:///etc/apt/aws-mirrors.txt jammy main restricted universe multiverse\n")
	sb.WriteString("deb mirror+file:///etc/apt/aws-mirrors.txt jammy-updates main restricted universe multiverse\n")
	sb.WriteString("deb mirror+file:///etc/apt/aws-mirrors.txt jammy-security main restricted universe multiverse\n")
	sb.WriteString("EOF\n\n")

	sb.WriteString("# Create mirror list with all AWS EC2 regional mirrors\n")
	sb.WriteString("cat > /etc/apt/aws-mirrors.txt << 'EOF'\n")
	sb.WriteString("http://us-west-2.ec2.archive.ubuntu.com/ubuntu/\n")
	sb.WriteString("http://us-west-1.ec2.archive.ubuntu.com/ubuntu/\n")
	sb.WriteString("http://us-east-1.ec2.archive.ubuntu.com/ubuntu/\n")
	sb.WriteString("http://us-east-2.ec2.archive.ubuntu.com/ubuntu/\n")
	sb.WriteString("EOF\n\n")

	// Update package lists
	sb.WriteString("# Update package lists\n")
	sb.WriteString("log_progress 'Refreshing package lists'\n")
	sb.WriteString("apt-get update -y\n")

	// Optionally run system upgrade
	if !skipSystemUpgrade {
		sb.WriteString("log_progress 'Upgrading system packages'\n")
		sb.WriteString("DEBIAN_FRONTEND=noninteractive apt-get upgrade -y\n")
	} else {
		sb.WriteString("# Skipping apt-get upgrade for faster setup (--skip-system-upgrade=true)\n")
	}
	sb.WriteString("\n")

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

	// Install desktop environment (XFCE for full EWMH support and window management)
	sb.WriteString(dcv.GenerateDesktopInstallScript("xfce"))

	// Install QGIS and related packages
	sb.WriteString(generateQGISInstallScript(env))

	// Install NICE DCV Server
	sb.WriteString(dcv.GenerateDCVInstallScript(env.DCVConfig))

	// Setup DCV session auto-creator service (creates session on boot)
	// Virtual sessions don't persist across reboots, so we use a systemd service
	sb.WriteString(dcv.GenerateDCVSessionCreatorService(env.DCVConfig))

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

	// Install QGIS and packages with progress reporting
	if len(env.Packages) > 0 {
		sb.WriteString("# Install QGIS and related packages with progress feedback\n")
		sb.WriteString(fmt.Sprintf("TOTAL_PACKAGES=%d\n", len(env.Packages)))
		sb.WriteString("echo '0' > /var/log/qgis-install-progress.txt\n")
		sb.WriteString("chmod 644 /var/log/qgis-install-progress.txt\n\n")

		sb.WriteString("DEBIAN_FRONTEND=noninteractive apt-get install -y \\\n")
		sb.WriteString("  --fix-missing \\\n")
		sb.WriteString("  -o Dpkg::Options::=\"--force-confdef\" \\\n")
		sb.WriteString("  -o Dpkg::Options::=\"--force-confold\" \\\n")
		sb.WriteString("  -o Dpkg::Progress-Fancy=\"1\" \\\n")
		sb.WriteString("  -o Dpkg::Progress=\"1\" \\\n")
		for i, pkg := range env.Packages {
			if i == len(env.Packages)-1 {
				sb.WriteString("  " + pkg + " 2>&1 | tee -a /var/log/qgis-install.log | while read line; do\n")
				sb.WriteString("    if echo \"$line\" | grep -q 'Progress:'; then\n")
				sb.WriteString("      echo \"$line\" | sed -n 's/.*Progress: \\[\\([0-9]*\\)%\\].*/\\1/p' > /var/log/qgis-install-progress.txt 2>/dev/null || true\n")
				sb.WriteString("    fi\n")
				sb.WriteString("  done\n\n")
			} else {
				sb.WriteString("  " + pkg + " \\\n")
			}
		}
		sb.WriteString("echo '100' > /var/log/qgis-install-progress.txt\n")
		sb.WriteString("log_progress 'QGIS installation complete'\n\n")
	}

	// Create QGIS user directories before first launch
	// Fixes: GitHub Issue #57 - QGIS error "Can not make qgis.db private copy"
	sb.WriteString("# Pre-create QGIS user directories to prevent startup errors\n")
	sb.WriteString("log_progress 'Creating QGIS user directories'\n")
	sb.WriteString("mkdir -p /home/ubuntu/.local/share/QGIS/QGIS3\n")
	sb.WriteString("mkdir -p /home/ubuntu/.local/share/QGIS/QGIS3/profiles/default\n")
	sb.WriteString("chown -R ubuntu:ubuntu /home/ubuntu/.local\n\n")

	// Configure PolicyKit to allow ubuntu user to create color managed devices
	// Fixes: GitHub Issue #57 - Authentication dialog "create a color managed device"
	sb.WriteString("# Allow ubuntu user to create color managed devices without authentication\n")
	sb.WriteString("log_progress 'Configuring colord PolicyKit permissions'\n")
	sb.WriteString("cat > /etc/polkit-1/localauthority/50-local.d/45-allow-colord.pkla << 'EOF'\n")
	sb.WriteString("[Allow Color Manager for ubuntu user]\n")
	sb.WriteString("Identity=unix-user:ubuntu\n")
	sb.WriteString("Action=org.freedesktop.color-manager.create-device\n")
	sb.WriteString("ResultAny=no\n")
	sb.WriteString("ResultInactive=no\n")
	sb.WriteString("ResultActive=yes\n")
	sb.WriteString("EOF\n\n")

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

	sb.WriteString("# Create XFCE autostart directory\n")
	sb.WriteString("mkdir -p /home/ubuntu/.config/autostart\n")
	sb.WriteString("mkdir -p /home/ubuntu/.local/bin\n\n")

	sb.WriteString("# Create QGIS fullscreen launcher script\n")
	sb.WriteString("cat > /home/ubuntu/.local/bin/qgis-fullscreen-launcher.sh << 'QGISLAUNCHER'\n")
	sb.WriteString("#!/bin/bash\n")
	sb.WriteString("# Wait for desktop to fully initialize\n")
	sb.WriteString("sleep 5\n")
	sb.WriteString("\n")
	sb.WriteString("# Launch QGIS in background\n")
	sb.WriteString("qgis &\n")
	sb.WriteString("QGIS_PID=$!\n")
	sb.WriteString("\n")
	sb.WriteString("# Wait for QGIS window to appear and make it fullscreen\n")
	sb.WriteString("for i in {1..20}; do\n")
	sb.WriteString("  sleep 1\n")
	sb.WriteString("  QGIS_WIN=$(wmctrl -l | grep -i qgis | head -1 | awk '{print $1}')\n")
	sb.WriteString("  if [ -n \"$QGIS_WIN\" ]; then\n")
	sb.WriteString("    # Found QGIS window - make it fullscreen\n")
	sb.WriteString("    wmctrl -i -r $QGIS_WIN -b add,fullscreen\n")
	sb.WriteString("    # Remove window decorations\n")
	sb.WriteString("    wmctrl -i -r $QGIS_WIN -b add,above\n")
	sb.WriteString("    break\n")
	sb.WriteString("  fi\n")
	sb.WriteString("done\n")
	sb.WriteString("QGISLAUNCHER\n\n")

	sb.WriteString("chmod +x /home/ubuntu/.local/bin/qgis-fullscreen-launcher.sh\n\n")

	sb.WriteString("# Create XFCE autostart desktop entry for QGIS\n")
	sb.WriteString("cat > /home/ubuntu/.config/autostart/qgis-fullscreen.desktop << 'QGISAUTOSTART'\n")
	sb.WriteString("[Desktop Entry]\n")
	sb.WriteString("Type=Application\n")
	sb.WriteString("Name=QGIS Fullscreen\n")
	sb.WriteString("Exec=/home/ubuntu/.local/bin/qgis-fullscreen-launcher.sh\n")
	sb.WriteString("Hidden=false\n")
	sb.WriteString("NoDisplay=false\n")
	sb.WriteString("X-GNOME-Autostart-enabled=true\n")
	sb.WriteString("QGISAUTOSTART\n\n")

	sb.WriteString("# Completely disable GNOME keyring\n")
	sb.WriteString("echo 'export GNOME_KEYRING_CONTROL=' >> /home/ubuntu/.profile\n")
	sb.WriteString("echo 'export GNOME_KEYRING_PID=' >> /home/ubuntu/.profile\n")
	sb.WriteString("echo 'export GNOME_KEYRING_CONTROL=' >> /home/ubuntu/.bashrc\n")
	sb.WriteString("echo 'export GNOME_KEYRING_PID=' >> /home/ubuntu/.bashrc\n")
	sb.WriteString("# Disable keyring autostart\n")
	sb.WriteString("mkdir -p /home/ubuntu/.config/autostart\n")
	sb.WriteString("cat > /home/ubuntu/.config/autostart/gnome-keyring-pkcs11.desktop << 'EOF'\n")
	sb.WriteString("[Desktop Entry]\n")
	sb.WriteString("Hidden=true\n")
	sb.WriteString("EOF\n")
	sb.WriteString("cat > /home/ubuntu/.config/autostart/gnome-keyring-secrets.desktop << 'EOF'\n")
	sb.WriteString("[Desktop Entry]\n")
	sb.WriteString("Hidden=true\n")
	sb.WriteString("EOF\n")
	sb.WriteString("cat > /home/ubuntu/.config/autostart/gnome-keyring-ssh.desktop << 'EOF'\n")
	sb.WriteString("[Desktop Entry]\n")
	sb.WriteString("Hidden=true\n")
	sb.WriteString("EOF\n\n")

	sb.WriteString("# Disable XFCE screensaver permanently\n")
	sb.WriteString("mkdir -p /home/ubuntu/.config/xfce4/xfconf/xfce-perchannel-xml\n")
	sb.WriteString("cat > /home/ubuntu/.config/xfce4/xfconf/xfce-perchannel-xml/xfce4-screensaver.xml << 'EOF'\n")
	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	sb.WriteString("<channel name=\"xfce4-screensaver\" version=\"1.0\">\n")
	sb.WriteString("  <property name=\"saver\" type=\"empty\">\n")
	sb.WriteString("    <property name=\"enabled\" type=\"bool\" value=\"false\"/>\n")
	sb.WriteString("  </property>\n")
	sb.WriteString("  <property name=\"lock\" type=\"empty\">\n")
	sb.WriteString("    <property name=\"enabled\" type=\"bool\" value=\"false\"/>\n")
	sb.WriteString("  </property>\n")
	sb.WriteString("</channel>\n")
	sb.WriteString("EOF\n")
	sb.WriteString("# Disable screensaver autostart\n")
	sb.WriteString("cat > /home/ubuntu/.config/autostart/xfce4-screensaver.desktop << 'EOF'\n")
	sb.WriteString("[Desktop Entry]\n")
	sb.WriteString("Hidden=true\n")
	sb.WriteString("EOF\n\n")

	sb.WriteString("# Disable XFCE power manager (not needed on cloud instances)\n")
	sb.WriteString("cat > /home/ubuntu/.config/autostart/xfce4-power-manager.desktop << 'EOF'\n")
	sb.WriteString("[Desktop Entry]\n")
	sb.WriteString("Hidden=true\n")
	sb.WriteString("EOF\n\n")

	sb.WriteString("# Set proper ownership\n")
	sb.WriteString("chown -R ubuntu:ubuntu /home/ubuntu/.config\n\n")

	sb.WriteString("# Create desktop shortcut\n")
	sb.WriteString("mkdir -p /home/ubuntu/Desktop\n")
	sb.WriteString("cp /usr/share/applications/org.qgis.qgis.desktop /home/ubuntu/Desktop/qgis.desktop 2>/dev/null || true\n")
	sb.WriteString("chmod +x /home/ubuntu/Desktop/qgis.desktop 2>/dev/null || true\n")
	sb.WriteString("chown ubuntu:ubuntu /home/ubuntu/Desktop/qgis.desktop 2>/dev/null || true\n\n")

	return sb.String()
}
