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
	sb.WriteString("authentication=\"none\"\n")
	sb.WriteString("\n")
	sb.WriteString("[session-management]\n")
	sb.WriteString("idle-timeout=0\n")
	sb.WriteString("max-concurrent-clients=1\n")
	sb.WriteString("\n")
	sb.WriteString("[session-management/automatic-console-session]\n")
	sb.WriteString(fmt.Sprintf("owner=\"%s\"\n", cfg.Owner))
	sb.WriteString("\n")
	sb.WriteString("[display]\n")
	sb.WriteString("enable-cu-desktops=true\n")
	sb.WriteString("\n")
	sb.WriteString("[display/linux]\n")
	sb.WriteString("use-layout-manager=true\n")
	sb.WriteString("web-client-max-head-resolution=(1920, 1080)\n")

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

	return sb.String()
}

// GenerateDCVSessionCreatorService generates a systemd service that auto-creates DCV sessions on boot
// This ensures virtual sessions persist across reboots
func GenerateDCVSessionCreatorService(cfg *Config) string {
	var sb strings.Builder

	sb.WriteString("# Setup DCV session auto-creator service\n")
	sb.WriteString("log_progress 'Configuring DCV session auto-creator'\n\n")

	// Create DCV init script that will start the desktop environment
	sb.WriteString("# Create DCV session init script\n")
	sb.WriteString("mkdir -p /opt/dcv-session\n")
	sb.WriteString(fmt.Sprintf("cat > /opt/dcv-session/init-%s.sh << 'DCVINIT'\n", cfg.SessionName))
	sb.WriteString("#!/bin/bash\n")
	sb.WriteString("# DCV session init script - starts desktop environment and applications\n")
	sb.WriteString("exec &>> /var/log/dcv-session-init.log\n")
	sb.WriteString("echo \"[$(date)] DCV session init starting...\"\n\n")
	sb.WriteString("# Start XFCE4 session in background\n")
	sb.WriteString("startxfce4 &\n")
	sb.WriteString("XFCE_PID=$!\n")
	sb.WriteString("echo \"[$(date)] Started XFCE4 session (PID: $XFCE_PID)\"\n\n")
	sb.WriteString("# Wait for XFCE to initialize and window manager to be ready\n")
	sb.WriteString("for i in {1..30}; do\n")
	sb.WriteString("  if wmctrl -m &>/dev/null; then\n")
	sb.WriteString("    echo \"[$(date)] Window manager is ready (attempt $i)\"\n")
	sb.WriteString("    break\n")
	sb.WriteString("  fi\n")
	sb.WriteString("  sleep 1\n")
	sb.WriteString("done\n\n")
	sb.WriteString("echo \"[$(date)] DCV session init complete - XFCE is running\"\n")
	sb.WriteString("wait $XFCE_PID\n")
	sb.WriteString("DCVINIT\n\n")
	sb.WriteString(fmt.Sprintf("chmod +x /opt/dcv-session/init-%s.sh\n\n", cfg.SessionName))

	// Create wait-for-dcv-ready script
	sb.WriteString("# Create script to wait for DCV server to be fully ready\n")
	sb.WriteString("cat > /usr/local/bin/wait-for-dcv-ready.sh << 'WAITSCRIPT'\n")
	sb.WriteString("#!/bin/bash\n")
	sb.WriteString("# Wait for DCV server to be fully ready before creating session\n")
	sb.WriteString("for i in {1..60}; do\n")
	sb.WriteString("  if /usr/bin/dcv list-sessions &>/dev/null; then\n")
	sb.WriteString("    echo \"[$(date)] DCV server is ready (attempt $i)\"\n")
	sb.WriteString("    exit 0\n")
	sb.WriteString("  fi\n")
	sb.WriteString("  echo \"[$(date)] Waiting for DCV server (attempt $i)...\"\n")
	sb.WriteString("  sleep 2\n")
	sb.WriteString("done\n")
	sb.WriteString("echo \"[$(date)] ERROR: DCV server not ready after 120 seconds\"\n")
	sb.WriteString("exit 1\n")
	sb.WriteString("WAITSCRIPT\n\n")
	sb.WriteString("chmod +x /usr/local/bin/wait-for-dcv-ready.sh\n\n")

	// Create systemd service that uses the init script
	sb.WriteString("cat > /etc/systemd/system/dcv-session-creator.service << 'SESSIONSERVICE'\n")
	sb.WriteString("[Unit]\n")
	sb.WriteString("Description=DCV Session Auto-Creator\n")
	sb.WriteString("After=dcvserver.service\n")
	sb.WriteString("Requires=dcvserver.service\n")
	sb.WriteString("\n")
	sb.WriteString("[Service]\n")
	sb.WriteString("Type=oneshot\n")
	sb.WriteString("RemainAfterExit=yes\n")
	sb.WriteString("User=root\n")
	sb.WriteString("ExecStartPre=/usr/local/bin/wait-for-dcv-ready.sh\n")
	sb.WriteString(fmt.Sprintf("ExecStart=/usr/bin/dcv create-session --type=%s --owner %s --init /opt/dcv-session/init-%s.sh %s\n",
		cfg.SessionType, cfg.Owner, cfg.SessionName, cfg.SessionName))
	sb.WriteString("\n")
	sb.WriteString("[Install]\n")
	sb.WriteString("WantedBy=multi-user.target\n")
	sb.WriteString("SESSIONSERVICE\n\n")

	sb.WriteString("# Enable the session creator service\n")
	sb.WriteString("systemctl daemon-reload\n")
	sb.WriteString("systemctl enable dcv-session-creator.service\n\n")

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
		sb.WriteString("  lightdm \\\n")
		sb.WriteString("  wmctrl\n\n")
	case "mate":
		sb.WriteString("DEBIAN_FRONTEND=noninteractive apt-get install -y \\\n")
		sb.WriteString("  mate-desktop-environment-core \\\n")
		sb.WriteString("  lightdm\n\n")
	case "minimal":
		sb.WriteString("DEBIAN_FRONTEND=noninteractive apt-get install -y \\\n")
		sb.WriteString("  xserver-xorg \\\n")
		sb.WriteString("  openbox \\\n")
		sb.WriteString("  lightdm \\\n")
		sb.WriteString("  dbus-x11 \\\n")
		sb.WriteString("  wmctrl\n\n")
	case "ultra-minimal":
		// Ultra-minimal: Just X server core + basic window manager for DCV
		// No display manager - DCV handles session management
		// Perfect for single-application remote desktop use
		sb.WriteString("DEBIAN_FRONTEND=noninteractive apt-get install -y \\\n")
		sb.WriteString("  xserver-xorg-core \\\n")
		sb.WriteString("  xserver-xorg-video-dummy \\\n")
		sb.WriteString("  x11-xserver-utils \\\n")
		sb.WriteString("  x11-utils \\\n")
		sb.WriteString("  openbox \\\n")
		sb.WriteString("  dbus-x11 \\\n")
		sb.WriteString("  wmctrl \\\n")
		sb.WriteString("  fonts-dejavu-core\n\n")
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
	sb.WriteString("IDLE_START=0\n")
	sb.WriteString("\n")
	sb.WriteString("while true; do\n")
	sb.WriteString("  # Check if DCV session exists\n")
	sb.WriteString("  if dcv list-sessions | grep -q \"$SESSION_NAME\"; then\n")
	sb.WriteString("    # Check for active connections\n")
	sb.WriteString("    CONNECTIONS=$(dcv list-connections -session $SESSION_NAME 2>/dev/null | wc -l)\n")
	sb.WriteString("    \n")
	sb.WriteString("    if [ $CONNECTIONS -eq 0 ]; then\n")
	sb.WriteString("      # No connections - start or continue idle timer\n")
	sb.WriteString("      if [ $IDLE_START -eq 0 ]; then\n")
	sb.WriteString("        IDLE_START=$(date +%s)\n")
	sb.WriteString("        logger \"DCV idle period started\"\n")
	sb.WriteString("      fi\n")
	sb.WriteString("      \n")
	sb.WriteString("      CURRENT_TIME=$(date +%s)\n")
	sb.WriteString("      IDLE_DURATION=$((CURRENT_TIME - IDLE_START))\n")
	sb.WriteString("      REMAINING=$((IDLE_TIMEOUT - IDLE_DURATION))\n")
	sb.WriteString("      \n")
	sb.WriteString("      if [ $IDLE_DURATION -ge $IDLE_TIMEOUT ]; then\n")
	sb.WriteString("        logger \"DCV idle timeout reached after ${IDLE_DURATION}s, shutting down\"\n")
	sb.WriteString("        shutdown -h now\n")
	sb.WriteString("      else\n")
	sb.WriteString("        logger \"DCV idle: ${IDLE_DURATION}s elapsed, ${REMAINING}s remaining\"\n")
	sb.WriteString("      fi\n")
	sb.WriteString("    else\n")
	sb.WriteString("      # Active connections - reset idle timer\n")
	sb.WriteString("      if [ $IDLE_START -ne 0 ]; then\n")
	sb.WriteString("        logger \"DCV connection detected, resetting idle timer\"\n")
	sb.WriteString("        IDLE_START=0\n")
	sb.WriteString("      fi\n")
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
