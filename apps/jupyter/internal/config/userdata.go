package config

import (
	"encoding/base64"
	"fmt"
	"strings"

	pkgconfig "github.com/scttfrdmn/aws-ide/pkg/config"
)

// GenerateUserData creates a cloud-init user data script for the given environment
func GenerateUserData(env *pkgconfig.Environment, idleTimeoutSeconds int, s3Bucket, s3SyncPath string) (string, error) {
	script := generateUserDataScript(env, idleTimeoutSeconds, s3Bucket, s3SyncPath)
	// AWS expects user data to be base64 encoded
	encoded := base64.StdEncoding.EncodeToString([]byte(script))
	return encoded, nil
}

// generateUserDataScript creates the actual bash script
func generateUserDataScript(env *pkgconfig.Environment, idleTimeoutSeconds int, s3Bucket, s3SyncPath string) string {
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

	sb.WriteString("log_progress 'Starting aws-jupyter environment setup'\n")
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

	// GPU detection and driver installation
	sb.WriteString(generateGPUSetupScript())

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

	// Setup Python and pip
	sb.WriteString("# Ensure pip is up to date\n")
	sb.WriteString("python3 -m pip install --upgrade pip setuptools wheel\n\n")

	// Install Python packages
	if len(env.PipPackages) > 0 {
		sb.WriteString("# Install Python packages\n")
		sb.WriteString("log_progress 'Installing Python packages'\n")
		sb.WriteString("python3 -m pip install \\\n")
		for i, pkg := range env.PipPackages {
			if i == len(env.PipPackages)-1 {
				sb.WriteString("  " + pkg + "\n\n")
			} else {
				sb.WriteString("  " + pkg + " \\\n")
			}
		}
	}

	// Install R packages if any
	if len(env.RPackages) > 0 {
		sb.WriteString("# Install R packages\n")
		sb.WriteString("log_progress 'Installing R packages'\n")
		sb.WriteString("R -e \"install.packages(c(")
		for i, pkg := range env.RPackages {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("'" + pkg + "'")
		}
		sb.WriteString("), repos='https://cloud.r-project.org/', dependencies=TRUE)\"\n\n")

		// Install IRkernel if present
		hasIRkernel := false
		for _, pkg := range env.RPackages {
			if pkg == "IRkernel" {
				hasIRkernel = true
				break
			}
		}
		if hasIRkernel {
			sb.WriteString("# Configure IRkernel for Jupyter\n")
			sb.WriteString("R -e \"IRkernel::installspec(user = FALSE)\"\n\n")
		}
	}

	// Install Julia and packages if any
	if len(env.JuliaPackages) > 0 {
		sb.WriteString("# Install Julia\n")
		sb.WriteString("log_progress 'Installing Julia'\n")
		sb.WriteString("wget -q https://julialang-s3.julialang.org/bin/linux/aarch64/1.10/julia-1.10.5-linux-aarch64.tar.gz -O /tmp/julia.tar.gz\n")
		sb.WriteString("tar -xzf /tmp/julia.tar.gz -C /opt/\n")
		sb.WriteString("ln -s /opt/julia-1.10.5/bin/julia /usr/local/bin/julia\n")
		sb.WriteString("rm /tmp/julia.tar.gz\n\n")

		sb.WriteString("# Install Julia packages\n")
		sb.WriteString("echo 'Installing Julia packages...'\n")
		sb.WriteString("sudo -u ubuntu julia -e 'using Pkg; ")
		for i, pkg := range env.JuliaPackages {
			if i > 0 {
				sb.WriteString("; ")
			}
			sb.WriteString("Pkg.add(\"" + pkg + "\")")
		}
		sb.WriteString("'\n\n")

		// Configure IJulia if present
		hasIJulia := false
		for _, pkg := range env.JuliaPackages {
			if pkg == "IJulia" {
				hasIJulia = true
				break
			}
		}
		if hasIJulia {
			sb.WriteString("# Configure IJulia kernel for Jupyter (system-wide)\n")
			sb.WriteString("sudo -u ubuntu julia -e 'using IJulia; installkernel(\"Julia\", \"--user\")'\n")
			sb.WriteString("sudo cp -r /home/ubuntu/.local/share/jupyter/kernels/julia-* /usr/local/share/jupyter/kernels/ 2>/dev/null || true\n\n")
		}
	}

	// Setup jupyter directory and permissions
	sb.WriteString("# Setup Jupyter workspace\n")
	sb.WriteString("mkdir -p /home/ubuntu/notebooks\n")
	sb.WriteString("chown -R ubuntu:ubuntu /home/ubuntu/notebooks\n\n")

	// Set environment variables
	if len(env.EnvironmentVars) > 0 {
		sb.WriteString("# Set environment variables\n")
		sb.WriteString("cat >> /home/ubuntu/.bashrc << 'EOF'\n")
		for key, value := range env.EnvironmentVars {
			sb.WriteString("export " + key + "=\"" + value + "\"\n")
		}
		sb.WriteString("EOF\n\n")
	}

	// Generate Jupyter config
	sb.WriteString("# Configure Jupyter Lab\n")
	sb.WriteString("mkdir -p /home/ubuntu/.jupyter\n")
	sb.WriteString("cat > /home/ubuntu/.jupyter/jupyter_lab_config.py << 'EOF'\n")
	sb.WriteString("c.ServerApp.ip = '0.0.0.0'\n")
	sb.WriteString("c.ServerApp.port = 8888\n")
	sb.WriteString("c.ServerApp.open_browser = False\n")
	sb.WriteString("c.ServerApp.allow_root = False\n")
	sb.WriteString("c.ServerApp.token = ''\n")
	sb.WriteString("c.ServerApp.password = ''\n")
	sb.WriteString("c.ServerApp.allow_origin = '*'\n")
	sb.WriteString("c.ServerApp.disable_check_xsrf = False\n")
	sb.WriteString("EOF\n")
	sb.WriteString("chown ubuntu:ubuntu /home/ubuntu/.jupyter/jupyter_lab_config.py\n\n")

	// Create systemd service for Jupyter
	sb.WriteString("# Create Jupyter systemd service\n")
	sb.WriteString("cat > /etc/systemd/system/jupyter.service << 'EOF'\n")
	sb.WriteString("[Unit]\n")
	sb.WriteString("Description=Jupyter Lab\n")
	sb.WriteString("After=network.target\n\n")
	sb.WriteString("[Service]\n")
	sb.WriteString("Type=simple\n")
	sb.WriteString("User=ubuntu\n")
	sb.WriteString("Environment=HOME=/home/ubuntu\n")
	sb.WriteString("WorkingDirectory=/home/ubuntu/notebooks\n")
	sb.WriteString("ExecStart=/usr/local/bin/jupyter lab\n")
	sb.WriteString("Restart=on-failure\n")
	sb.WriteString("RestartSec=10\n\n")
	sb.WriteString("[Install]\n")
	sb.WriteString("WantedBy=multi-user.target\n")
	sb.WriteString("EOF\n\n")

	// Enable and start Jupyter service
	sb.WriteString("# Enable and start Jupyter\n")
	sb.WriteString("log_progress 'Starting Jupyter Lab service'\n")
	sb.WriteString("systemctl daemon-reload\n")
	sb.WriteString("systemctl enable jupyter.service\n")
	sb.WriteString("systemctl start jupyter.service\n\n")

	// Setup idle detection system
	sb.WriteString("# Setup idle detection and auto-stop system\n")
	sb.WriteString("echo 'Setting up idle detection...'\n\n")

	// Install jq for JSON parsing
	sb.WriteString("apt-get install -y jq ec2-instance-connect\n\n")

	// Embed the idle monitor script
	sb.WriteString(generateIdleMonitorScript())

	// Embed the auto-stop script
	sb.WriteString(generateAutoStopScript())

	// Create systemd service files
	sb.WriteString(generateIdleDetectionServices(idleTimeoutSeconds))

	// Enable and start the services
	sb.WriteString("systemctl daemon-reload\n")
	sb.WriteString("systemctl enable jupyter-idle-monitor.timer\n")
	sb.WriteString("systemctl enable jupyter-auto-stop.timer\n")
	sb.WriteString("systemctl start jupyter-idle-monitor.timer\n")
	sb.WriteString("systemctl start jupyter-auto-stop.timer\n")
	sb.WriteString("echo 'Idle detection system installed and enabled'\n\n")

	// Setup S3 data sync if bucket is specified
	if s3Bucket != "" {
		sb.WriteString("# Setup S3 data sync\n")
		sb.WriteString("log_progress 'Setting up S3 data sync'\n")
		sb.WriteString(generateS3SyncScript(s3Bucket, s3SyncPath))
		sb.WriteString("\n")
	}

	// Final status
	sb.WriteString("log_progress 'Setup complete - Jupyter Lab is ready'\n")
	sb.WriteString("echo 'COMPLETE' >> $PROGRESS_LOG\n")
	sb.WriteString("echo 'aws-jupyter environment setup complete!'\n")
	sb.WriteString("echo 'Jupyter Lab is running on port 8888'\n")
	sb.WriteString("echo 'Use Session Manager or SSH tunnel to connect'\n")

	return sb.String()
}

// GetRawUserData returns the user data script without base64 encoding (for debugging)
func GetRawUserData(env *pkgconfig.Environment, idleTimeoutSeconds int, s3Bucket, s3SyncPath string) string {
	return generateUserDataScript(env, idleTimeoutSeconds, s3Bucket, s3SyncPath)
}

// generateIdleMonitorScript creates the idle monitor script
func generateIdleMonitorScript() string {
	return `# Create idle monitor script
cat > /usr/local/bin/jupyter-idle-monitor.sh << 'IDLE_MONITOR_EOF'
#!/bin/bash
# Jupyter Idle Monitor
set -e

# Configuration
JUPYTER_PORT="${JUPYTER_PORT:-8888}"
IDLE_STATE_FILE="/var/run/jupyter-idle-status"
LAST_ACTIVITY_FILE="/var/run/jupyter-last-activity"
CPU_THRESHOLD="${CPU_THRESHOLD:-10}"
LOG_FILE="/var/log/jupyter-idle-monitor.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

check_jupyter_kernels() {
    local jupyter_url="http://localhost:${JUPYTER_PORT}"
    local kernels_response
    if ! kernels_response=$(curl -s -f "${jupyter_url}/api/kernels" 2>/dev/null); then
        log "WARNING: Could not connect to Jupyter API"
        return 1
    fi
    local kernel_count=$(echo "$kernels_response" | jq '. | length' 2>/dev/null || echo "0")
    if [ "$kernel_count" -gt 0 ]; then
        log "Active kernels detected: $kernel_count"
        local busy_count=$(echo "$kernels_response" | jq '[.[] | select(.execution_state == "busy")] | length' 2>/dev/null || echo "0")
        if [ "$busy_count" -gt 0 ]; then
            log "Busy kernels detected: $busy_count"
            return 0
        fi
    fi
    local sessions_response
    if sessions_response=$(curl -s -f "${jupyter_url}/api/sessions" 2>/dev/null); then
        local session_count=$(echo "$sessions_response" | jq '. | length' 2>/dev/null || echo "0")
        if [ "$session_count" -gt 0 ]; then
            log "Active sessions detected: $session_count"
            return 0
        fi
    fi
    log "No active Jupyter kernels or sessions"
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
    local jupyter_pids=$(pgrep -f "jupyter-lab" || echo "")
    if [ -z "$jupyter_pids" ]; then
        log "WARNING: Jupyter process not found"
        return 1
    fi
    for pid in $jupyter_pids; do
        local children=$(pgrep -P "$pid" || echo "")
        for child in $children; do
            local cpu_usage=$(ps -p "$child" -o %cpu= 2>/dev/null | awk '{print int($1)}')
            if [ -n "$cpu_usage" ] && [ "$cpu_usage" -gt 5 ]; then
                local cmd=$(ps -p "$child" -o comm= 2>/dev/null)
                log "Active computation process detected: PID=$child CMD=$cmd CPU=${cpu_usage}%"
                return 0
            fi
        done
    done
    log "No active computation processes"
    return 1
}

check_network_activity() {
    log "Skipping network connection check (using Jupyter API instead)"
    return 1
}

main() {
    log "=== Starting idle detection check ==="
    local is_active=0
    if check_jupyter_kernels; then
        is_active=1
    elif check_cpu_usage; then
        is_active=1
    elif check_running_processes; then
        is_active=1
    elif check_network_activity; then
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

chmod +x /usr/local/bin/jupyter-idle-monitor.sh

`
}

// generateAutoStopScript creates the auto-stop script
func generateAutoStopScript() string {
	return `# Create auto-stop script
cat > /usr/local/bin/jupyter-auto-stop.sh << 'AUTO_STOP_EOF'
#!/bin/bash
# Jupyter Auto-Stop Service
set -e

# Configuration
IDLE_STATE_FILE="/var/run/jupyter-idle-status"
LAST_ACTIVITY_FILE="/var/run/jupyter-last-activity"
IDLE_TIMEOUT="${IDLE_TIMEOUT:-14400}"
IDLE_ACTION="${IDLE_ACTION:-stop}"
LOG_FILE="/var/log/jupyter-auto-stop.log"
ENABLED_FILE="/etc/jupyter-auto-stop.enabled"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

if [ ! -f "$ENABLED_FILE" ]; then
    if [ ! -f "/var/run/jupyter-auto-stop-initialized" ]; then
        echo "enabled" > "$ENABLED_FILE"
        touch "/var/run/jupyter-auto-stop-initialized"
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

chmod +x /usr/local/bin/jupyter-auto-stop.sh

`
}

// generateIdleDetectionServices creates the systemd service and timer files
func generateIdleDetectionServices(idleTimeoutSeconds int) string {
	idleTimeoutEnv := fmt.Sprintf("Environment=\"IDLE_TIMEOUT=%d\"", idleTimeoutSeconds)

	return fmt.Sprintf(`# Create idle monitor systemd service
cat > /etc/systemd/system/jupyter-idle-monitor.service << 'SERVICE_EOF'
[Unit]
Description=Jupyter Idle Monitor
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/jupyter-idle-monitor.sh
Environment="JUPYTER_PORT=8888"
Environment="CPU_THRESHOLD=10"
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
SERVICE_EOF

# Create idle monitor timer
cat > /etc/systemd/system/jupyter-idle-monitor.timer << 'TIMER_EOF'
[Unit]
Description=Run Jupyter idle monitor every 5 minutes
Requires=jupyter-idle-monitor.service

[Timer]
OnBootSec=5min
OnUnitActiveSec=5min
AccuracySec=1min

[Install]
WantedBy=timers.target
TIMER_EOF

# Create auto-stop systemd service
cat > /etc/systemd/system/jupyter-auto-stop.service << 'SERVICE_EOF'
[Unit]
Description=Jupyter Auto-Stop Service
After=network.target jupyter-idle-monitor.service

[Service]
Type=oneshot
ExecStart=/usr/local/bin/jupyter-auto-stop.sh
%s
Environment="IDLE_ACTION=stop"
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
SERVICE_EOF

# Create auto-stop timer
cat > /etc/systemd/system/jupyter-auto-stop.timer << 'TIMER_EOF'
[Unit]
Description=Check for auto-stop every minute
Requires=jupyter-auto-stop.service

[Timer]
OnBootSec=1min
OnUnitActiveSec=1min
AccuracySec=30s

[Install]
WantedBy=timers.target
TIMER_EOF

`, idleTimeoutEnv)
}

// generateS3SyncScript creates the S3 mounting script using mountpoint-s3
func generateS3SyncScript(s3Bucket, s3SyncPath string) string {
	return fmt.Sprintf(`# Install and configure mountpoint-s3 for S3 data sync
cat > /tmp/setup-s3-sync.sh << 'S3_SYNC_EOF'
#!/bin/bash
set -e

# Configuration
S3_BUCKET="%s"
MOUNT_PATH="%s"
LOG_FILE="/var/log/s3-sync-setup.log"

log() {
    echo "[$(date '+%%Y-%%m-%%d %%H:%%M:%%S')] $1" | tee -a "$LOG_FILE"
}

log "=== Starting S3 sync setup ==="
log "S3 Bucket: $S3_BUCKET"
log "Mount Path: $MOUNT_PATH"

# Detect architecture
ARCH=$(dpkg --print-architecture)
log "Detected architecture: $ARCH"

# Install mountpoint-s3
log "Installing mountpoint-s3..."
if [ "$ARCH" = "arm64" ]; then
    wget -q https://s3.amazonaws.com/mountpoint-s3-release/latest/arm64/mount-s3.deb -O /tmp/mount-s3.deb
else
    wget -q https://s3.amazonaws.com/mountpoint-s3-release/latest/x64/mount-s3.deb -O /tmp/mount-s3.deb
fi

dpkg -i /tmp/mount-s3.deb || apt-get install -f -y
rm /tmp/mount-s3.deb
log "mountpoint-s3 installed successfully"

# Verify installation
if ! command -v mount-s3 &> /dev/null; then
    log "ERROR: mount-s3 command not found after installation"
    exit 1
fi

# Create mount point directory
log "Creating mount point directory: $MOUNT_PATH"
mkdir -p "$MOUNT_PATH"
chown ubuntu:ubuntu "$MOUNT_PATH"

# Test S3 access
log "Testing S3 bucket access..."
if aws s3 ls "s3://$S3_BUCKET" > /dev/null 2>&1; then
    log "S3 bucket access verified"
else
    log "WARNING: Unable to access S3 bucket. Check IAM permissions."
    log "Required permissions: s3:ListBucket, s3:GetObject, s3:PutObject, s3:DeleteObject"
    exit 1
fi

# Create systemd mount unit
log "Creating systemd mount unit..."
MOUNT_UNIT_NAME=$(echo "$MOUNT_PATH" | sed 's/\//-/g' | sed 's/^-//')
cat > /etc/systemd/system/${MOUNT_UNIT_NAME}.mount << EOF
[Unit]
Description=Mount S3 bucket ${S3_BUCKET} to ${MOUNT_PATH}
After=network-online.target
Wants=network-online.target

[Mount]
What=${S3_BUCKET}
Where=${MOUNT_PATH}
Type=fuse.mount-s3
Options=allow-delete,allow-other,uid=$(id -u ubuntu),gid=$(id -g ubuntu),region=${AWS_REGION:-us-west-2}

[Install]
WantedBy=multi-user.target
EOF

# Enable and start the mount
log "Enabling and starting S3 mount..."
systemctl daemon-reload
systemctl enable ${MOUNT_UNIT_NAME}.mount
systemctl start ${MOUNT_UNIT_NAME}.mount

# Wait for mount to be ready
sleep 5

# Verify mount
if mountpoint -q "$MOUNT_PATH"; then
    log "S3 bucket mounted successfully at $MOUNT_PATH"
    log "Mount status:"
    df -h "$MOUNT_PATH" | tee -a "$LOG_FILE"
else
    log "ERROR: Failed to mount S3 bucket"
    systemctl status ${MOUNT_UNIT_NAME}.mount | tee -a "$LOG_FILE"
    exit 1
fi

# Create a README in the mount point
cat > ${MOUNT_PATH}/.README << 'README_EOF'
This directory is mounted from S3 bucket: ${S3_BUCKET}

Files written here are automatically synced to S3.
Files are cached locally for performance.

Notes:
- Changes are visible immediately on this instance
- Other instances accessing the same bucket will see changes
- Use standard file operations (cp, mv, rm)
- Large files are streamed rather than cached entirely

For more information: https://github.com/awslabs/mountpoint-s3
README_EOF

chown ubuntu:ubuntu ${MOUNT_PATH}/.README

log "=== S3 sync setup complete ==="
log "You can now use $MOUNT_PATH for your data"
S3_SYNC_EOF

chmod +x /tmp/setup-s3-sync.sh
/tmp/setup-s3-sync.sh
rm /tmp/setup-s3-sync.sh

`, s3Bucket, s3SyncPath)
}

// generateGPUSetupScript creates a script to detect and configure NVIDIA GPUs
func generateGPUSetupScript() string {
	return `# GPU detection and driver installation
cat > /tmp/setup-gpu.sh << 'GPU_SETUP_EOF'
#!/bin/bash
set -e

LOG_FILE="/var/log/gpu-setup.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log "=== Starting GPU detection and setup ==="

# Check if instance has GPU by checking for NVIDIA devices
if ! lspci | grep -i nvidia > /dev/null 2>&1; then
    log "No NVIDIA GPU detected. Skipping GPU setup."
    exit 0
fi

log "NVIDIA GPU detected! Installing drivers and CUDA toolkit..."

# Get GPU info
GPU_INFO=$(lspci | grep -i nvidia)
log "GPU Info: $GPU_INFO"

# Install NVIDIA driver repository
log "Adding NVIDIA driver repository..."
apt-get install -y software-properties-common
add-apt-repository -y ppa:graphics-drivers/ppa
apt-get update -y

# Install NVIDIA driver (535 series for Ubuntu 24.04)
log "Installing NVIDIA driver..."
DEBIAN_FRONTEND=noninteractive apt-get install -y nvidia-driver-535 nvidia-utils-535

# Install CUDA toolkit (12.2 for latest compatibility)
log "Installing CUDA toolkit..."
wget -q https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2404/x86_64/cuda-keyring_1.1-1_all.deb
dpkg -i cuda-keyring_1.1-1_all.deb
apt-get update -y
DEBIAN_FRONTEND=noninteractive apt-get install -y cuda-toolkit-12-2

# Set up environment variables
log "Configuring CUDA environment variables..."
cat >> /etc/environment << 'ENV_EOF'
CUDA_HOME=/usr/local/cuda
PATH=/usr/local/cuda/bin:$PATH
LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH
ENV_EOF

# Also add to ubuntu user's bashrc
cat >> /home/ubuntu/.bashrc << 'BASHRC_EOF'
export CUDA_HOME=/usr/local/cuda
export PATH=/usr/local/cuda/bin:$PATH
export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH
BASHRC_EOF

# Create symlink for CUDA if needed
if [ ! -L /usr/local/cuda ]; then
    ln -s /usr/local/cuda-12.2 /usr/local/cuda
fi

log "GPU drivers and CUDA toolkit installed successfully"
log "Note: A reboot is required for GPU drivers to be fully active"
log "nvidia-smi will be available after the instance restarts"

log "=== GPU setup complete ==="
GPU_SETUP_EOF

chmod +x /tmp/setup-gpu.sh
/tmp/setup-gpu.sh || echo "GPU setup encountered an error, continuing with instance launch..."
rm /tmp/setup-gpu.sh

`
}
