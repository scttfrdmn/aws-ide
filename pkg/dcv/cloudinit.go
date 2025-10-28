package dcv

import (
	"fmt"
	"strings"
)

// CloudInitOptions holds options for generating DCV cloud-init scripts
type CloudInitOptions struct {
	DCVConfig           *Config
	DesktopEnvironment  string   // gnome, xfce, minimal
	RequiresGPU         bool     // Install GPU drivers
	PreInstallScript    string   // Custom script to run before DCV install
	PostInstallScript   string   // Custom script to run after DCV install
	ApplicationPackages []string // Packages to install for the application
	IdleTimeoutSeconds  int      // Auto-shutdown timeout
}

// GenerateDCVInstallScript generates the DCV server installation portion of cloud-init
func GenerateDCVInstallScript(cfg *Config) string {
	var sb strings.Builder

	sb.WriteString("# Install NICE DCV Server\n")
	sb.WriteString("log_progress 'Installing NICE DCV Server'\n\n")

	// Download and install DCV
	sb.WriteString("cd /tmp\n")
	sb.WriteString("wget -q https://d1uj6qtbmh3dt5.cloudfront.net/nice-dcv-ubuntu2204-x86_64.tgz\n")
	sb.WriteString("tar -xzf nice-dcv-ubuntu2204-x86_64.tgz\n")
	sb.WriteString("cd nice-dcv-*-ubuntu2204-x86_64\n\n")

	sb.WriteString("# Install DCV packages\n")
	sb.WriteString("DEBIAN_FRONTEND=noninteractive apt-get install -y ./nice-dcv-server_*.deb\n")
	sb.WriteString("DEBIAN_FRONTEND=noninteractive apt-get install -y ./nice-dcv-web-viewer_*.deb\n")
	sb.WriteString("DEBIAN_FRONTEND=noninteractive apt-get install -y ./nice-xdcv_*.deb\n\n")

	// Configure DCV
	sb.WriteString("# Configure DCV Server\n")
	sb.WriteString("cat > /etc/dcv/dcv.conf << 'DCVCONF'\n")
	sb.WriteString("[connectivity]\n")
	sb.WriteString(fmt.Sprintf("web-port=%d\n", cfg.Port))
	sb.WriteString("web-use-https=true\n")
	sb.WriteString("\n")
	sb.WriteString("[security]\n")
	sb.WriteString("authentication=\"system\"\n")
	sb.WriteString("\n")
	sb.WriteString("[session-management/automatic-console-session]\n")
	sb.WriteString(fmt.Sprintf("owner=\"%s\"\n", cfg.Owner))
	sb.WriteString("\n")
	sb.WriteString("[display]\n")
	sb.WriteString("enable-cu-desktops=true\n")

	if cfg.EnableGPU {
		sb.WriteString("\n")
		sb.WriteString("[gpu]\n")
		sb.WriteString("enable-gpu=true\n")
	}

	sb.WriteString("DCVCONF\n\n")

	// Enable and start DCV
	sb.WriteString("# Enable and start DCV service\n")
	sb.WriteString("systemctl enable dcvserver\n")
	sb.WriteString("systemctl start dcvserver\n\n")

	// Set owner password for DCV login
	sb.WriteString(fmt.Sprintf("# Set %s user password for DCV login\n", cfg.Owner))
	sb.WriteString(fmt.Sprintf("echo '%s:$(openssl rand -base64 12)' | chpasswd\n\n", cfg.Owner))

	return sb.String()
}

// GenerateDCVSessionScript generates the script to create a DCV session
func GenerateDCVSessionScript(cfg *Config) string {
	var sb strings.Builder

	sb.WriteString("# Create DCV Session\n")
	sb.WriteString("log_progress 'Creating DCV session'\n\n")

	sb.WriteString("# Wait for DCV server to be ready\n")
	sb.WriteString("for i in {1..30}; do\n")
	sb.WriteString("  if systemctl is-active --quiet dcvserver; then\n")
	sb.WriteString("    break\n")
	sb.WriteString("  fi\n")
	sb.WriteString("  sleep 2\n")
	sb.WriteString("done\n\n")

	sb.WriteString("# Create virtual session\n")
	sb.WriteString(fmt.Sprintf("dcv create-session --type=%s --owner %s %s\n\n",
		cfg.SessionType, cfg.Owner, cfg.SessionName))

	return sb.String()
}

// GenerateDesktopInstallScript generates the desktop environment installation
func GenerateDesktopInstallScript(desktopType string) string {
	var sb strings.Builder

	sb.WriteString("# Install Desktop Environment\n")
	sb.WriteString(fmt.Sprintf("log_progress 'Installing %s desktop environment'\n", desktopType))

	switch desktopType {
	case "gnome":
		sb.WriteString("DEBIAN_FRONTEND=noninteractive apt-get install -y \\\n")
		sb.WriteString("  ubuntu-desktop \\\n")
		sb.WriteString("  gnome-session \\\n")
		sb.WriteString("  gdm3\n\n")
	case "xfce":
		sb.WriteString("DEBIAN_FRONTEND=noninteractive apt-get install -y \\\n")
		sb.WriteString("  xubuntu-desktop \\\n")
		sb.WriteString("  xfce4 \\\n")
		sb.WriteString("  xfce4-goodies \\\n")
		sb.WriteString("  lightdm\n\n")
	case "mate":
		sb.WriteString("DEBIAN_FRONTEND=noninteractive apt-get install -y \\\n")
		sb.WriteString("  mate-desktop-environment-core \\\n")
		sb.WriteString("  lightdm\n\n")
	case "minimal":
		sb.WriteString("DEBIAN_FRONTEND=noninteractive apt-get install -y \\\n")
		sb.WriteString("  xserver-xorg \\\n")
		sb.WriteString("  openbox \\\n")
		sb.WriteString("  lightdm\n\n")
	default:
		// Default to XFCE for unknown types
		sb.WriteString("DEBIAN_FRONTEND=noninteractive apt-get install -y \\\n")
		sb.WriteString("  xubuntu-desktop \\\n")
		sb.WriteString("  xfce4\n\n")
	}

	sb.WriteString("systemctl set-default graphical.target\n\n")

	return sb.String()
}

// GenerateGPUSetupScript generates GPU driver installation script
func GenerateGPUSetupScript() string {
	var sb strings.Builder

	sb.WriteString("# GPU Setup\n")
	sb.WriteString("log_progress 'Detecting GPU'\n")
	sb.WriteString("if lspci | grep -i nvidia; then\n")
	sb.WriteString("  log_progress 'NVIDIA GPU detected, installing drivers'\n")
	sb.WriteString("  \n")
	sb.WriteString("  # Install NVIDIA drivers\n")
	sb.WriteString("  apt-get install -y linux-headers-$(uname -r)\n")
	sb.WriteString("  DEBIAN_FRONTEND=noninteractive apt-get install -y nvidia-driver-535\n")
	sb.WriteString("  \n")
	sb.WriteString("  # Install CUDA toolkit\n")
	sb.WriteString("  wget -q https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/cuda-keyring_1.0-1_all.deb\n")
	sb.WriteString("  dpkg -i cuda-keyring_1.0-1_all.deb\n")
	sb.WriteString("  apt-get update\n")
	sb.WriteString("  DEBIAN_FRONTEND=noninteractive apt-get install -y cuda-toolkit-12-2\n")
	sb.WriteString("  \n")
	sb.WriteString("  # Add CUDA to PATH\n")
	sb.WriteString("  echo 'export PATH=/usr/local/cuda/bin:$PATH' >> /etc/profile.d/cuda.sh\n")
	sb.WriteString("  echo 'export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH' >> /etc/profile.d/cuda.sh\n")
	sb.WriteString("  \n")
	sb.WriteString("  log_progress 'GPU drivers installed'\n")
	sb.WriteString("else\n")
	sb.WriteString("  log_progress 'No GPU detected, skipping GPU setup'\n")
	sb.WriteString("fi\n\n")

	return sb.String()
}

// GenerateIdleMonitorScript generates idle timeout monitoring
func GenerateIdleMonitorScript(cfg *Config, idleTimeoutSeconds int) string {
	var sb strings.Builder

	sb.WriteString("# Setup idle timeout monitoring\n")
	sb.WriteString(fmt.Sprintf("log_progress 'Configuring auto-stop after %d seconds of idle'\n\n", idleTimeoutSeconds))

	sb.WriteString("# Create idle monitor script\n")
	sb.WriteString("cat > /usr/local/bin/dcv-idle-monitor.sh << 'IDLESCRIPT'\n")
	sb.WriteString("#!/bin/bash\n")
	sb.WriteString(fmt.Sprintf("IDLE_TIMEOUT=%d\n", idleTimeoutSeconds))
	sb.WriteString(fmt.Sprintf("SESSION_NAME=\"%s\"\n", cfg.SessionName))
	sb.WriteString("\n")
	sb.WriteString("while true; do\n")
	sb.WriteString("  # Check if DCV session exists\n")
	sb.WriteString("  if dcv list-sessions | grep -q \"$SESSION_NAME\"; then\n")
	sb.WriteString("    # Check for active connections\n")
	sb.WriteString("    CONNECTIONS=$(dcv list-connections -session $SESSION_NAME 2>/dev/null | wc -l)\n")
	sb.WriteString("    if [ $CONNECTIONS -eq 0 ]; then\n")
	sb.WriteString("      logger \"DCV idle timeout reached, shutting down\"\n")
	sb.WriteString("      shutdown -h now\n")
	sb.WriteString("    fi\n")
	sb.WriteString("  fi\n")
	sb.WriteString("  sleep 60\n")
	sb.WriteString("done\n")
	sb.WriteString("IDLESCRIPT\n\n")

	sb.WriteString("chmod +x /usr/local/bin/dcv-idle-monitor.sh\n\n")

	sb.WriteString("# Create systemd service for idle monitor\n")
	sb.WriteString("cat > /etc/systemd/system/dcv-idle-monitor.service << 'IDLESERVICE'\n")
	sb.WriteString("[Unit]\n")
	sb.WriteString("Description=DCV Idle Monitor\n")
	sb.WriteString("After=dcvserver.service\n")
	sb.WriteString("\n")
	sb.WriteString("[Service]\n")
	sb.WriteString("Type=simple\n")
	sb.WriteString("ExecStart=/usr/local/bin/dcv-idle-monitor.sh\n")
	sb.WriteString("Restart=on-failure\n")
	sb.WriteString("\n")
	sb.WriteString("[Install]\n")
	sb.WriteString("WantedBy=multi-user.target\n")
	sb.WriteString("IDLESERVICE\n\n")

	sb.WriteString("systemctl daemon-reload\n")
	sb.WriteString("systemctl enable dcv-idle-monitor.service\n")
	sb.WriteString("systemctl start dcv-idle-monitor.service\n\n")

	return sb.String()
}
