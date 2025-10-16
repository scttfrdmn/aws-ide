package config

import (
	"encoding/base64"
	"strings"
)

// GenerateUserData creates a cloud-init user data script for the given environment
func GenerateUserData(env *Environment) (string, error) {
	script := generateUserDataScript(env)
	// AWS expects user data to be base64 encoded
	encoded := base64.StdEncoding.EncodeToString([]byte(script))
	return encoded, nil
}

// generateUserDataScript creates the actual bash script
func generateUserDataScript(env *Environment) string {
	var sb strings.Builder

	// Start with bash shebang and error handling
	sb.WriteString("#!/bin/bash\n")
	sb.WriteString("set -e\n")
	sb.WriteString("set -x\n\n")

	// Log file for debugging
	sb.WriteString("exec > >(tee /var/log/user-data.log)\n")
	sb.WriteString("exec 2>&1\n\n")

	sb.WriteString("echo 'Starting aws-jupyter environment setup'\n")
	sb.WriteString("echo 'Environment: " + env.Name + "'\n\n")

	// Update system
	sb.WriteString("# Update system packages\n")
	sb.WriteString("apt-get update -y\n")
	sb.WriteString("apt-get upgrade -y\n\n")

	// Install SSM Agent
	sb.WriteString("# Install AWS Systems Manager Agent\n")
	sb.WriteString("echo 'Installing SSM Agent...'\n")
	sb.WriteString("snap install amazon-ssm-agent --classic\n")
	sb.WriteString("systemctl enable snap.amazon-ssm-agent.amazon-ssm-agent.service\n")
	sb.WriteString("systemctl start snap.amazon-ssm-agent.amazon-ssm-agent.service\n\n")

	// Install system packages
	if len(env.Packages) > 0 {
		sb.WriteString("# Install system packages\n")
		sb.WriteString("echo 'Installing system packages...'\n")
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

	// Install R packages if any
	if len(env.RPackages) > 0 {
		sb.WriteString("# Install R packages\n")
		sb.WriteString("echo 'Installing R packages...'\n")
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
		sb.WriteString("echo 'Installing Julia...'\n")
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
	sb.WriteString("systemctl daemon-reload\n")
	sb.WriteString("systemctl enable jupyter.service\n")
	sb.WriteString("systemctl start jupyter.service\n\n")

	// Setup idle detection script (placeholder for future implementation)
	sb.WriteString("# Setup idle detection (future implementation)\n")
	sb.WriteString("# This will monitor Jupyter activity and shutdown if idle\n\n")

	// Final status
	sb.WriteString("echo 'aws-jupyter environment setup complete!'\n")
	sb.WriteString("echo 'Jupyter Lab is running on port 8888'\n")
	sb.WriteString("echo 'Use Session Manager or SSH tunnel to connect'\n")

	return sb.String()
}

// GetRawUserData returns the user data script without base64 encoding (for debugging)
func GetRawUserData(env *Environment) string {
	return generateUserDataScript(env)
}
