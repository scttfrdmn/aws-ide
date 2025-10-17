#!/bin/bash
# Jupyter Auto-Stop Service
# Stops or hibernates the instance after prolonged idle period

set -e

# Configuration
IDLE_STATE_FILE="/var/run/jupyter-idle-status"
LAST_ACTIVITY_FILE="/var/run/jupyter-last-activity"
IDLE_TIMEOUT="${IDLE_TIMEOUT:-14400}"  # 4 hours default (in seconds)
IDLE_ACTION="${IDLE_ACTION:-stop}"  # stop or hibernate
LOG_FILE="/var/log/jupyter-auto-stop.log"
ENABLED_FILE="/etc/jupyter-auto-stop.enabled"

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Check if auto-stop is enabled
if [ ! -f "$ENABLED_FILE" ]; then
    # Check if this is first run - create enabled file with default enabled
    if [ ! -f "/var/run/jupyter-auto-stop-initialized" ]; then
        echo "enabled" > "$ENABLED_FILE"
        touch "/var/run/jupyter-auto-stop-initialized"
        log "Auto-stop initialized and enabled"
    fi
fi

if [ -f "$ENABLED_FILE" ] && [ "$(cat $ENABLED_FILE)" = "disabled" ]; then
    # Silent exit when disabled
    exit 0
fi

# Check if required files exist
if [ ! -f "$IDLE_STATE_FILE" ]; then
    log "WARNING: Idle state file not found, skipping check"
    exit 0
fi

if [ ! -f "$LAST_ACTIVITY_FILE" ]; then
    log "WARNING: Last activity file not found, skipping check"
    exit 0
fi

# Read current idle state
CURRENT_STATE=$(cat "$IDLE_STATE_FILE")

if [ "$CURRENT_STATE" != "idle" ]; then
    # System is active, nothing to do
    exit 0
fi

# System is idle, check duration
LAST_ACTIVITY=$(cat "$LAST_ACTIVITY_FILE")
NOW=$(date +%s)
IDLE_DURATION=$((NOW - LAST_ACTIVITY))

log "System idle for ${IDLE_DURATION}s (threshold: ${IDLE_TIMEOUT}s)"

if [ $IDLE_DURATION -lt $IDLE_TIMEOUT ]; then
    # Not idle long enough
    REMAINING=$((IDLE_TIMEOUT - IDLE_DURATION))
    log "Time until auto-stop: ${REMAINING}s"
    exit 0
fi

# Idle timeout exceeded - prepare to stop
log "========================================"
log "IDLE TIMEOUT EXCEEDED - INITIATING SHUTDOWN"
log "========================================"
log "Idle duration: ${IDLE_DURATION}s"
log "Idle threshold: ${IDLE_TIMEOUT}s"
log "Action: ${IDLE_ACTION}"

# Get instance metadata
INSTANCE_ID=$(ec2-metadata --instance-id 2>/dev/null | cut -d' ' -f2 || echo "unknown")
REGION=$(ec2-metadata --availability-zone 2>/dev/null | cut -d' ' -f2 | sed 's/[a-z]$//' || echo "us-east-1")

log "Instance ID: $INSTANCE_ID"
log "Region: $REGION"

# Perform the action
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
