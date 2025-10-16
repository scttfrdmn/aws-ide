#!/bin/bash
# Jupyter Idle Monitor
# Checks multiple signals to determine if the system is idle

set -e

# Configuration
JUPYTER_PORT="${JUPYTER_PORT:-8888}"
JUPYTER_TOKEN="${JUPYTER_TOKEN:-}"
IDLE_STATE_FILE="/var/run/jupyter-idle-status"
LAST_ACTIVITY_FILE="/var/run/jupyter-last-activity"
CPU_THRESHOLD="${CPU_THRESHOLD:-10}"  # CPU usage percentage threshold
LOG_FILE="/var/log/jupyter-idle-monitor.log"

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Check if Jupyter kernels are active
check_jupyter_kernels() {
    local jupyter_url="http://localhost:${JUPYTER_PORT}"

    # Try to query kernels API
    local kernels_response
    if ! kernels_response=$(curl -s -f "${jupyter_url}/api/kernels" 2>/dev/null); then
        log "WARNING: Could not connect to Jupyter API"
        return 1  # Assume active if we can't check
    fi

    # Check if any kernels exist
    local kernel_count=$(echo "$kernels_response" | jq '. | length' 2>/dev/null || echo "0")
    if [ "$kernel_count" -gt 0 ]; then
        log "Active kernels detected: $kernel_count"

        # Check if any kernel is busy
        local busy_count=$(echo "$kernels_response" | jq '[.[] | select(.execution_state == "busy")] | length' 2>/dev/null || echo "0")
        if [ "$busy_count" -gt 0 ]; then
            log "Busy kernels detected: $busy_count"
            return 0  # Active
        fi

        # Check last activity time for each kernel
        local recent_activity=$(echo "$kernels_response" | jq '[.[] | select(.last_activity != null)] | length' 2>/dev/null || echo "0")
        if [ "$recent_activity" -gt 0 ]; then
            log "Kernels with recent activity: $recent_activity"
            return 0  # Active
        fi
    fi

    # Check sessions (notebooks with active kernels)
    local sessions_response
    if sessions_response=$(curl -s -f "${jupyter_url}/api/sessions" 2>/dev/null); then
        local session_count=$(echo "$sessions_response" | jq '. | length' 2>/dev/null || echo "0")
        if [ "$session_count" -gt 0 ]; then
            log "Active sessions detected: $session_count"
            return 0  # Active
        fi
    fi

    log "No active Jupyter kernels or sessions"
    return 1  # Idle
}

# Check CPU usage
check_cpu_usage() {
    # Get CPU idle percentage and calculate usage
    local cpu_idle=$(top -bn1 | grep "Cpu(s)" | sed "s/.*, *\([0-9.]*\)%* id.*/\1/" | awk '{print int($1)}')
    local cpu_usage=$((100 - cpu_idle))

    if [ "$cpu_usage" -gt "$CPU_THRESHOLD" ]; then
        log "CPU usage above threshold: ${cpu_usage}% > ${CPU_THRESHOLD}%"
        return 0  # Active
    fi

    log "CPU usage below threshold: ${cpu_usage}% <= ${CPU_THRESHOLD}%"
    return 1  # Idle
}

# Check for active computation processes
check_running_processes() {
    # Check for Python/R/Julia processes that are children of jupyter
    local jupyter_pids=$(pgrep -f "jupyter-lab" || echo "")

    if [ -z "$jupyter_pids" ]; then
        log "WARNING: Jupyter process not found"
        return 1  # Assume idle if jupyter not running
    fi

    # Check for child processes with significant CPU usage
    for pid in $jupyter_pids; do
        local children=$(pgrep -P "$pid" || echo "")
        for child in $children; then
            # Check if process is using CPU (via ps)
            local cpu_usage=$(ps -p "$child" -o %cpu= 2>/dev/null | awk '{print int($1)}')
            if [ -n "$cpu_usage" ] && [ "$cpu_usage" -gt 5 ]; then
                local cmd=$(ps -p "$child" -o comm= 2>/dev/null)
                log "Active computation process detected: PID=$child CMD=$cmd CPU=${cpu_usage}%"
                return 0  # Active
            fi
        done
    done

    log "No active computation processes"
    return 1  # Idle
}

# Check network activity to Jupyter
check_network_activity() {
    # Look for recent connections to Jupyter port (excluding this script)
    local recent_connections=$(ss -tn state established "( sport = :${JUPYTER_PORT} or dport = :${JUPYTER_PORT} )" 2>/dev/null | grep -v "State" | wc -l)

    if [ "$recent_connections" -gt 0 ]; then
        log "Active network connections to Jupyter: $recent_connections"
        return 0  # Active
    fi

    log "No active network connections to Jupyter"
    return 1  # Idle
}

# Main idle detection logic
main() {
    log "=== Starting idle detection check ==="

    local is_active=0

    # Check all signals (OR logic - any signal being active means system is active)
    if check_jupyter_kernels; then
        is_active=1
    elif check_cpu_usage; then
        is_active=1
    elif check_running_processes; then
        is_active=1
    elif check_network_activity; then
        is_active=1
    fi

    # Update state files
    if [ $is_active -eq 1 ]; then
        echo "active" > "$IDLE_STATE_FILE"
        date +%s > "$LAST_ACTIVITY_FILE"
        log "RESULT: System is ACTIVE"
    else
        echo "idle" > "$IDLE_STATE_FILE"
        # Don't update last activity time - keep the old one
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

# Run main function
main
