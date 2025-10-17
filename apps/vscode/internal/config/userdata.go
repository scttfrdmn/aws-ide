package config

import (
	"encoding/base64"
	"fmt"
	"strings"

	pkgconfig "github.com/scttfrdmn/aws-ide/pkg/config"
)

// GenerateUserData creates a cloud-init user data script for the given environment
func GenerateUserData(env *pkgconfig.Environment, idleTimeoutSeconds int) (string, error) {
	script := generateUserDataScript(env, idleTimeoutSeconds)
	// AWS expects user data to be base64 encoded
	encoded := base64.StdEncoding.EncodeToString([]byte(script))
	return encoded, nil
}

// generateUserDataScript creates the actual bash script for VSCode Server (code-server)
func generateUserDataScript(env *pkgconfig.Environment, idleTimeoutSeconds int) string {
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

	sb.WriteString("log_progress 'Starting aws-vscode environment setup'\n")
	sb.WriteString("echo 'Environment: " + env.Name + "'\n\n")

	// Update system
	sb.WriteString("# Update system packages\n")
	sb.WriteString("log_progress 'Updating system packages'\n")
	sb.WriteString("apt-get update -y\n")
	sb.WriteString("apt-get upgrade -y\n\n")

	// Install SSM Agent
	sb.WriteString("# Install AWS Systems Manager Agent\n")
	sb.WriteString("log_progress 'Installing SSM Agent'\n")
	sb.WriteString("snap install amazon-ssm-agent --classic\n")
	sb.WriteString("systemctl enable snap.amazon-ssm-agent.amazon-ssm-agent.service\n")
	sb.WriteString("systemctl start snap.amazon-ssm-agent.amazon-ssm-agent.service\n\n")

	// Install system packages
	if len(env.Packages) > 0 {
		sb.WriteString("# Install system packages\n")
		sb.WriteString("log_progress 'Installing system packages'\n")
		sb.WriteString("DEBIAN_FRONTEND=noninteractive apt-get install -y \\\n")
		for i, pkg := range env.Packages {
			if i == len(env.Packages)-1 {
				sb.WriteString("  " + pkg + "\n\n")
			} else {
				sb.WriteString("  " + pkg + " \\\n")
			}
		}
	}

	// Install code-server using official installation script
	sb.WriteString("# Install code-server\n")
	sb.WriteString("log_progress 'Installing code-server'\n")
	sb.WriteString("export HOME=/root\n")
	sb.WriteString("curl -fsSL https://code-server.dev/install.sh | sh\n\n")

	// Install Node.js if specified
	if nodeVersion, ok := env.EnvironmentVars["NODEJS_VERSION"]; ok {
		sb.WriteString("# Install Node.js " + nodeVersion + "\n")
		sb.WriteString("log_progress 'Installing Node.js " + nodeVersion + "'\n")
		sb.WriteString("curl -fsSL https://deb.nodesource.com/setup_" + nodeVersion + ".x | bash -\n")
		sb.WriteString("apt-get install -y nodejs\n")
		sb.WriteString("npm install -g yarn pnpm\n\n")
	}

	// Install Python if specified
	if pythonVersion, ok := env.EnvironmentVars["PYTHON_VERSION"]; ok && pythonVersion != "" {
		sb.WriteString("# Setup Python\n")
		sb.WriteString("echo 'Setting up Python...'\n")
		sb.WriteString("apt-get install -y python3-pip python3-venv python3-dev\n")
		sb.WriteString("python3 -m pip install --upgrade pip setuptools wheel\n\n")
	}

	// Install Python packages
	if len(env.PipPackages) > 0 {
		sb.WriteString("# Install Python packages\n")
		sb.WriteString("echo 'Installing Python packages...'\n")
		sb.WriteString("python3 -m pip install \\\n")
		for i, pkg := range env.PipPackages {
			if i == len(env.PipPackages)-1 {
				sb.WriteString("  " + pkg + "\n\n")
			} else {
				sb.WriteString("  " + pkg + " \\\n")
			}
		}
	}

	// Install Go if specified
	if goVersion, ok := env.EnvironmentVars["GO_VERSION"]; ok && goVersion != "" {
		sb.WriteString("# Install Go " + goVersion + "\n")
		sb.WriteString("echo 'Installing Go...'\n")
		// Detect architecture
		sb.WriteString("ARCH=$(dpkg --print-architecture)\n")
		sb.WriteString("if [ \"$ARCH\" = \"arm64\" ]; then\n")
		sb.WriteString("  GO_ARCH=\"arm64\"\n")
		sb.WriteString("else\n")
		sb.WriteString("  GO_ARCH=\"amd64\"\n")
		sb.WriteString("fi\n")
		sb.WriteString("wget -q https://go.dev/dl/go" + goVersion + ".linux-${GO_ARCH}.tar.gz -O /tmp/go.tar.gz\n")
		sb.WriteString("rm -rf /usr/local/go\n")
		sb.WriteString("tar -C /usr/local -xzf /tmp/go.tar.gz\n")
		sb.WriteString("rm /tmp/go.tar.gz\n")
		sb.WriteString("ln -sf /usr/local/go/bin/go /usr/local/bin/go\n")
		sb.WriteString("ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt\n\n")
	}

	// Install npm packages globally if any
	if npmPackages, ok := env.EnvironmentVars["NPM_PACKAGES"]; ok && npmPackages != "" {
		sb.WriteString("# Install global npm packages\n")
		sb.WriteString("echo 'Installing npm packages...'\n")
		sb.WriteString("npm install -g " + npmPackages + "\n\n")
	}

	// Setup workspace directory
	sb.WriteString("# Setup VSCode workspace\n")
	sb.WriteString("mkdir -p /home/ubuntu/workspace\n")
	sb.WriteString("chown -R ubuntu:ubuntu /home/ubuntu/workspace\n\n")

	// Set environment variables
	if len(env.EnvironmentVars) > 0 {
		sb.WriteString("# Set environment variables\n")
		sb.WriteString("cat >> /home/ubuntu/.bashrc << 'EOF'\n")
		for key, value := range env.EnvironmentVars {
			// Skip special vars that were already processed
			if key != "NODEJS_VERSION" && key != "PYTHON_VERSION" && key != "GO_VERSION" && key != "NPM_PACKAGES" && key != "VSCODE_EXTENSIONS" {
				sb.WriteString("export " + key + "=\"" + value + "\"\n")
			}
		}
		// Add Go to PATH if installed
		if _, ok := env.EnvironmentVars["GO_VERSION"]; ok {
			sb.WriteString("export PATH=$PATH:/usr/local/go/bin:/home/ubuntu/go/bin\n")
		}
		sb.WriteString("EOF\n\n")
	}

	// Configure code-server
	sb.WriteString("# Configure code-server\n")
	sb.WriteString("mkdir -p /home/ubuntu/.config/code-server\n")
	sb.WriteString("cat > /home/ubuntu/.config/code-server/config.yaml << 'EOF'\n")
	sb.WriteString("bind-addr: 0.0.0.0:8080\n")
	sb.WriteString("auth: password\n")
	sb.WriteString("password: " + generatePassword() + "\n")
	sb.WriteString("cert: false\n")
	sb.WriteString("EOF\n")
	sb.WriteString("chown -R ubuntu:ubuntu /home/ubuntu/.config/code-server\n\n")

	// Install VSCode extensions if specified
	if extensions, ok := env.EnvironmentVars["VSCODE_EXTENSIONS"]; ok && extensions != "" {
		sb.WriteString("# Install VSCode extensions\n")
		sb.WriteString("log_progress 'Installing VSCode extensions'\n")
		extList := strings.Split(extensions, ",")
		for _, ext := range extList {
			ext = strings.TrimSpace(ext)
			if ext != "" {
				sb.WriteString("sudo -u ubuntu code-server --install-extension " + ext + " || true\n")
			}
		}
		sb.WriteString("\n")
	}

	// Create systemd service for code-server
	sb.WriteString("# Create code-server systemd service\n")
	sb.WriteString("cat > /etc/systemd/system/code-server.service << 'EOF'\n")
	sb.WriteString("[Unit]\n")
	sb.WriteString("Description=code-server\n")
	sb.WriteString("After=network.target\n\n")
	sb.WriteString("[Service]\n")
	sb.WriteString("Type=simple\n")
	sb.WriteString("User=ubuntu\n")
	sb.WriteString("Environment=HOME=/home/ubuntu\n")
	sb.WriteString("WorkingDirectory=/home/ubuntu/workspace\n")
	sb.WriteString("ExecStart=/usr/bin/code-server\n")
	sb.WriteString("Restart=on-failure\n")
	sb.WriteString("RestartSec=10\n\n")
	sb.WriteString("[Install]\n")
	sb.WriteString("WantedBy=multi-user.target\n")
	sb.WriteString("EOF\n\n")

	// Enable and start code-server service
	sb.WriteString("# Enable and start code-server\n")
	sb.WriteString("log_progress 'Starting code-server service'\n")
	sb.WriteString("systemctl daemon-reload\n")
	sb.WriteString("systemctl enable code-server.service\n")
	sb.WriteString("systemctl start code-server.service\n\n")

	// Setup idle detection system
	sb.WriteString("# Setup idle detection and auto-stop system\n")
	sb.WriteString("echo 'Setting up idle detection...'\n\n")

	// Install jq and AWS CLI v2 for auto-stop functionality
	sb.WriteString("# Install dependencies\n")
	sb.WriteString("apt-get install -y jq ec2-instance-connect unzip\n\n")

	// Install AWS CLI v2
	sb.WriteString("# Install AWS CLI v2\n")
	sb.WriteString("echo 'Installing AWS CLI v2...'\n")
	sb.WriteString("ARCH=$(dpkg --print-architecture)\n")
	sb.WriteString("if [ \"$ARCH\" = \"arm64\" ]; then\n")
	sb.WriteString("  curl -fsSL \"https://awscli.amazonaws.com/awscli-exe-linux-aarch64.zip\" -o \"/tmp/awscliv2.zip\"\n")
	sb.WriteString("else\n")
	sb.WriteString("  curl -fsSL \"https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip\" -o \"/tmp/awscliv2.zip\"\n")
	sb.WriteString("fi\n")
	sb.WriteString("unzip -q /tmp/awscliv2.zip -d /tmp\n")
	sb.WriteString("/tmp/aws/install --update || /tmp/aws/install\n")
	sb.WriteString("rm -rf /tmp/awscliv2.zip /tmp/aws\n")
	sb.WriteString("aws --version\n\n")

	// Embed the idle monitor script
	sb.WriteString(generateIdleMonitorScript())

	// Embed the auto-stop script
	sb.WriteString(generateAutoStopScript())

	// Create systemd service files
	sb.WriteString(generateIdleDetectionServices(idleTimeoutSeconds))

	// Enable and start the services
	sb.WriteString("systemctl daemon-reload\n")
	sb.WriteString("systemctl enable vscode-idle-monitor.timer\n")
	sb.WriteString("systemctl enable vscode-auto-stop.timer\n")
	sb.WriteString("systemctl start vscode-idle-monitor.timer\n")
	sb.WriteString("systemctl start vscode-auto-stop.timer\n")
	sb.WriteString("echo 'Idle detection system installed and enabled'\n\n")

	// Final status
	sb.WriteString("log_progress 'Setup complete - VSCode Server is ready'\n")
	sb.WriteString("echo 'COMPLETE' >> $PROGRESS_LOG\n")
	sb.WriteString("echo 'aws-vscode environment setup complete!'\n")
	sb.WriteString("echo 'VSCode Server is running on port 8080'\n")
	sb.WriteString("echo 'Use Session Manager or SSH tunnel to connect'\n")

	return sb.String()
}

// GetRawUserData returns the user data script without base64 encoding (for debugging)
func GetRawUserData(env *pkgconfig.Environment, idleTimeoutSeconds int) string {
	return generateUserDataScript(env, idleTimeoutSeconds)
}

// generatePassword creates a simple password for code-server
func generatePassword() string {
	// For now, use a simple fixed password. In production, this should be generated randomly
	return "vscode2024"
}

// generateIdleMonitorScript creates the idle monitor script for VSCode
func generateIdleMonitorScript() string {
	return `# Create idle monitor script
cat > /usr/local/bin/vscode-idle-monitor.sh << 'IDLE_MONITOR_EOF'
#!/bin/bash
# VSCode Idle Monitor
set -e

# Configuration
IDLE_STATE_FILE="/var/run/vscode-idle-status"
LAST_ACTIVITY_FILE="/var/run/vscode-last-activity"
CPU_THRESHOLD="${CPU_THRESHOLD:-10}"
LOG_FILE="/var/log/vscode-idle-monitor.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

check_vscode_sessions() {
    # Check for active code-server sessions by looking at process activity
    local codeserver_pids=$(pgrep -f "code-server" || echo "")
    if [ -z "$codeserver_pids" ]; then
        log "WARNING: code-server process not found"
        return 1
    fi

    # Check for active Node.js processes (language servers, extensions)
    local node_count=$(pgrep -c "node" 2>/dev/null || echo "0")
    if [ "$node_count" -gt 3 ]; then  # More than just code-server itself
        log "Active VSCode sessions detected: $node_count node processes"
        return 0
    fi

    log "No active VSCode sessions"
    return 1
}

check_cpu_usage() {
    local cpu_idle=$(top -bn1 | grep "Cpu(s)" | sed "s/.*, *\([0-9.]*\)%* id.*/\1/" | awk '{print int($1)}')
    local cpu_usage=$((100 - cpu_idle))
    if [ "$cpu_usage" -gt "$CPU_THRESHOLD" ]; then
        log "CPU usage above threshold: ${cpu_usage}% > ${CPU_THRESHOLD}%"
        return 0
    fi
    log "CPU usage below threshold: ${cpu_usage}% <= ${CPU_THRESHOLD}%"
    return 1
}

check_running_processes() {
    # Check for active development processes (compilers, build tools, etc.)
    local dev_processes=$(pgrep -f "go build|npm|yarn|python|gcc|make" || echo "")
    if [ -n "$dev_processes" ]; then
        log "Active development processes detected"
        return 0
    fi
    log "No active development processes"
    return 1
}

main() {
    log "=== Starting idle detection check ==="
    local is_active=0
    if check_vscode_sessions; then
        is_active=1
    elif check_cpu_usage; then
        is_active=1
    elif check_running_processes; then
        is_active=1
    fi

    if [ $is_active -eq 1 ]; then
        echo "active" > "$IDLE_STATE_FILE"
        date +%s > "$LAST_ACTIVITY_FILE"
        log "RESULT: System is ACTIVE"
    else
        echo "idle" > "$IDLE_STATE_FILE"
        if [ ! -f "$LAST_ACTIVITY_FILE" ]; then
            date +%s > "$LAST_ACTIVITY_FILE"
        fi
        local last_activity=$(cat "$LAST_ACTIVITY_FILE")
        local now=$(date +%s)
        local idle_duration=$((now - last_activity))
        log "RESULT: System is IDLE (duration: ${idle_duration}s)"
    fi
    log "=== Idle detection check complete ==="
}

main
IDLE_MONITOR_EOF

chmod +x /usr/local/bin/vscode-idle-monitor.sh

`
}

// generateAutoStopScript creates the auto-stop script
func generateAutoStopScript() string {
	return `# Create auto-stop script
cat > /usr/local/bin/vscode-auto-stop.sh << 'AUTO_STOP_EOF'
#!/bin/bash
# VSCode Auto-Stop Service
set -e

# Configuration
IDLE_STATE_FILE="/var/run/vscode-idle-status"
LAST_ACTIVITY_FILE="/var/run/vscode-last-activity"
IDLE_TIMEOUT="${IDLE_TIMEOUT:-14400}"
IDLE_ACTION="${IDLE_ACTION:-stop}"
LOG_FILE="/var/log/vscode-auto-stop.log"
ENABLED_FILE="/etc/vscode-auto-stop.enabled"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

if [ ! -f "$ENABLED_FILE" ]; then
    if [ ! -f "/var/run/vscode-auto-stop-initialized" ]; then
        echo "enabled" > "$ENABLED_FILE"
        touch "/var/run/vscode-auto-stop-initialized"
        log "Auto-stop initialized and enabled"
    fi
fi

if [ -f "$ENABLED_FILE" ] && [ "$(cat $ENABLED_FILE)" = "disabled" ]; then
    exit 0
fi

if [ ! -f "$IDLE_STATE_FILE" ] || [ ! -f "$LAST_ACTIVITY_FILE" ]; then
    exit 0
fi

CURRENT_STATE=$(cat "$IDLE_STATE_FILE")
if [ "$CURRENT_STATE" != "idle" ]; then
    exit 0
fi

LAST_ACTIVITY=$(cat "$LAST_ACTIVITY_FILE")
NOW=$(date +%s)
IDLE_DURATION=$((NOW - LAST_ACTIVITY))

log "System idle for ${IDLE_DURATION}s (threshold: ${IDLE_TIMEOUT}s)"

if [ $IDLE_DURATION -lt $IDLE_TIMEOUT ]; then
    REMAINING=$((IDLE_TIMEOUT - IDLE_DURATION))
    log "Time until auto-stop: ${REMAINING}s"
    exit 0
fi

log "========================================"
log "IDLE TIMEOUT EXCEEDED - INITIATING SHUTDOWN"
log "========================================"
log "Idle duration: ${IDLE_DURATION}s"
log "Idle threshold: ${IDLE_TIMEOUT}s"
log "Action: ${IDLE_ACTION}"

INSTANCE_ID=$(ec2-metadata --instance-id 2>/dev/null | cut -d' ' -f2 || echo "unknown")
REGION=$(ec2-metadata --availability-zone 2>/dev/null | cut -d' ' -f2 | sed 's/[a-z]$//' || echo "us-east-1")

log "Instance ID: $INSTANCE_ID"
log "Region: $REGION"

if [ "$IDLE_ACTION" = "hibernate" ]; then
    log "Attempting to hibernate instance..."
    if aws ec2 stop-instances --instance-ids "$INSTANCE_ID" --region "$REGION" --hibernate 2>&1 | tee -a "$LOG_FILE"; then
        log "Hibernate command sent successfully"
    else
        log "ERROR: Failed to hibernate, falling back to stop"
        aws ec2 stop-instances --instance-ids "$INSTANCE_ID" --region "$REGION" 2>&1 | tee -a "$LOG_FILE" || log "ERROR: Stop command also failed"
    fi
else
    log "Stopping instance..."
    if aws ec2 stop-instances --instance-ids "$INSTANCE_ID" --region "$REGION" 2>&1 | tee -a "$LOG_FILE"; then
        log "Stop command sent successfully"
    else
        log "ERROR: Failed to stop instance"
        exit 1
    fi
fi

log "Auto-stop process complete"
log "========================================"
AUTO_STOP_EOF

chmod +x /usr/local/bin/vscode-auto-stop.sh

`
}

// generateIdleDetectionServices creates the systemd service and timer files
func generateIdleDetectionServices(idleTimeoutSeconds int) string {
	idleTimeoutEnv := fmt.Sprintf("Environment=\"IDLE_TIMEOUT=%d\"", idleTimeoutSeconds)

	return fmt.Sprintf(`# Create idle monitor systemd service
cat > /etc/systemd/system/vscode-idle-monitor.service << 'SERVICE_EOF'
[Unit]
Description=VSCode Idle Monitor
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/vscode-idle-monitor.sh
Environment="CPU_THRESHOLD=10"
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
SERVICE_EOF

# Create idle monitor timer
cat > /etc/systemd/system/vscode-idle-monitor.timer << 'TIMER_EOF'
[Unit]
Description=Run VSCode idle monitor every 5 minutes
Requires=vscode-idle-monitor.service

[Timer]
OnBootSec=5min
OnUnitActiveSec=5min
AccuracySec=1min

[Install]
WantedBy=timers.target
TIMER_EOF

# Create auto-stop systemd service
cat > /etc/systemd/system/vscode-auto-stop.service << 'SERVICE_EOF'
[Unit]
Description=VSCode Auto-Stop Service
After=network.target vscode-idle-monitor.service

[Service]
Type=oneshot
ExecStart=/usr/local/bin/vscode-auto-stop.sh
%s
Environment="IDLE_ACTION=stop"
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
SERVICE_EOF

# Create auto-stop timer
cat > /etc/systemd/system/vscode-auto-stop.timer << 'TIMER_EOF'
[Unit]
Description=Check for auto-stop every minute
Requires=vscode-auto-stop.service

[Timer]
OnBootSec=1min
OnUnitActiveSec=1min
AccuracySec=30s

[Install]
WantedBy=timers.target
TIMER_EOF

`, idleTimeoutEnv)
}
