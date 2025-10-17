# aws-rstudio

[![Go Version](https://img.shields.io/badge/go-1.22+-blue.svg)](https://golang.org/dl/)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A powerful CLI tool for launching secure RStudio Server instances on AWS EC2 Graviton processors with professional-grade networking and security features.

**Full lifecycle management** with commands for launching, connecting, stopping, and terminating instances across public and private subnets with Session Manager or SSH access.

> **Note**: This project is part of the [AWS IDE monorepo](../../README.md). It shares infrastructure with [aws-jupyter](../jupyter/) and other IDE launchers.

## 🚀 Key Features

### **🔒 Security & Access Control**
- **Session Manager**: Secure access without SSH keys or bastion hosts
- **Traditional SSH**: Full SSH key management with economical reuse strategy
- **Private Subnets**: Enterprise-grade isolation with optional NAT Gateway
- **Smart Security Groups**: Automatic firewall rules with IP restrictions

### **🏗️ Infrastructure Management**
- **Zero Infrastructure**: Deploy from your laptop with full AWS integration
- **Graviton Optimized**: ARM64 instances for best price/performance ratio
- **Advanced Networking**: Public/private subnet support with NAT Gateway creation
- **Resource Reuse**: Smart reuse of existing infrastructure to minimize costs

### **⚙️ Environment System**
- **Built-in Environments**: Pre-configured R environments for different use cases
- **YAML Configuration**: Simple, version-controlled environment definitions
- **Package Management**: Automatic handling of system packages, R packages, and dependencies

### **💰 Cost Optimization**
- **Automatic Idle Detection**: Multi-signal monitoring (CPU, processes)
- **Auto-Stop**: Configurable idle timeout to prevent runaway costs
- **Flexible Timeouts**: Set custom idle timeouts (e.g., `--idle-timeout 30m`, `2h`, `8h`)

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/scttfrdmn/aws-ide
cd aws-ide/apps/rstudio

# Build
go build -o aws-rstudio ./cmd/aws-rstudio

# Install
sudo mv aws-rstudio /usr/local/bin/
```

### Verify Installation

```bash
aws-rstudio --version
aws-rstudio --help
```

## AWS Authentication

Before using aws-rstudio, configure AWS credentials:

```bash
# Option 1: AWS CLI (Recommended)
aws configure

# Option 2: Environment variables
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_DEFAULT_REGION="us-east-1"

# Option 3: AWS profiles
aws configure --profile myprofile
aws-rstudio launch --profile myprofile
```

See the [AWS Authentication Guide](../jupyter/docs/AWS_AUTHENTICATION.md) for detailed setup instructions.

## Quick Start

### **⚡ Simplest Launch (5 Seconds)**

```bash
# Launch with all defaults - perfect for getting started
aws-rstudio launch

# The CLI will:
# ✓ Configure IAM roles and security groups
# ✓ Launch a t4g.medium Graviton instance
# ✓ Install RStudio Server with minimal R environment
# ✓ Auto-stop after 4 hours of inactivity (saves money!)
# ✓ Show you the RStudio URL when ready (http://...)
```

### **🔐 Secure Launch (Session Manager - Recommended)**

```bash
# Most secure: Private subnet with Session Manager
aws-rstudio launch --connection session-manager --subnet-type private --create-nat-gateway

# Secure and cost-effective: Private subnet without internet
aws-rstudio launch --connection session-manager --subnet-type private

# Session Manager with public subnet (no SSH exposure)
aws-rstudio launch --connection session-manager
```

### **🔑 Traditional SSH Launch**

```bash
# Standard SSH with public subnet
aws-rstudio launch --connection ssh

# SSH with custom environment and instance type
aws-rstudio launch --env tidyverse --instance-type t4g.large --connection ssh
```

### **💰 Cost Control with Idle Detection**

```bash
# Auto-stop after 30 minutes of inactivity (great for testing)
aws-rstudio launch --idle-timeout 30m

# Custom timeout for long-running work
aws-rstudio launch --idle-timeout 8h

# Auto-stop detects:
# ✓ Active R sessions and computation processes
# ✓ High CPU usage (>10% threshold)
# ✓ Running RStudio processes
# ✓ Instance automatically stops when idle to save costs
```

### **🛠️ Instance & Resource Management**

```bash
# Instance lifecycle
aws-rstudio list                        # Show all instances with status
aws-rstudio status i-0abc123def         # Detailed instance information
aws-rstudio connect i-0abc123def        # Connect to existing instance
aws-rstudio stop i-0abc123def           # Stop instance (preserves EBS)
aws-rstudio terminate i-0abc123def      # Terminate instance (cleanup)

# SSH key management
aws-rstudio key list                    # View local and AWS key pairs
aws-rstudio key show                    # Show default key details
aws-rstudio key validate                # Check key file permissions
```

### **📋 Preview Changes (Dry Run)**

```bash
# Always preview before launching
aws-rstudio launch --dry-run --connection session-manager --subnet-type private --create-nat-gateway
```

## Environments

Built-in R environments:
- `minimal`: Basic R installation with rmarkdown (t4g.small, 20GB)
- `tidyverse`: Modern data science with tidyverse, dplyr, ggplot2 (t4g.medium, 30GB)
- `bioconductor`: Bioinformatics with DESeq2, edgeR, GenomicRanges (t4g.large, 50GB)
- `shiny`: Interactive web applications with Shiny, plotly, leaflet (t4g.medium, 30GB)

### Generating Custom Environments

Create custom R environments from your local setup:

```bash
# List available environments
aws-rstudio env list

# View environment details
aws-rstudio env show tidyverse

# Generate custom environment (coming soon)
aws-rstudio generate --name my-rproject --source ./research
```

## 🔗 Connection Methods

### **Session Manager (Recommended)**
- **No SSH keys required** - eliminates key management complexity
- **Enhanced security** - access through AWS SSM, no direct internet exposure
- **Audit logging** - all sessions logged in CloudTrail
- **Works anywhere** - no bastion hosts or VPN required

See the [Session Manager Setup Guide](../jupyter/docs/SESSION_MANAGER_SETUP.md) for installation instructions.

```bash
# Launch with Session Manager
aws-rstudio launch --connection session-manager

# Connect to running instance
aws-rstudio connect i-0abc123def
# or use AWS CLI directly:
aws ssm start-session --target i-0abc123def --profile myprofile
```

**Prerequisites**: Requires AWS CLI and Session Manager plugin installed.

### **Traditional SSH**
- **Full SSH access** - direct SSH connection with automatic key management
- **Economical key reuse** - smart naming strategy (aws-rstudio-{region})
- **Secure local storage** - private keys stored with 600 permissions
- **IP restrictions** - security groups restrict access to your current IP

```bash
# Launch with SSH
aws-rstudio launch --connection ssh

# Connect directly
ssh -i ~/.aws-rstudio/keys/aws-rstudio-us-west-2.pem ubuntu@1.2.3.4
```

## 🌐 Networking Options

### **Public Subnets** (Default)
- Direct internet access for package installations
- Public IP assigned automatically
- Best for development and testing

### **Private Subnets** (Enterprise)
- Enhanced security with no direct internet exposure
- Requires NAT Gateway for internet access (additional cost ~$45/month)
- Ideal for production and sensitive workloads

See the [Private Subnet Guide](../jupyter/docs/PRIVATE_SUBNET_GUIDE.md) for best practices.

```bash
# Private subnet with internet access (recommended for production)
aws-rstudio launch --subnet-type private --create-nat-gateway

# Private subnet without internet (cost-effective but limited functionality)
aws-rstudio launch --subnet-type private

# Cost breakdown displayed during dry-run
aws-rstudio launch --dry-run --subnet-type private --create-nat-gateway
```

## ⚙️ Configuration

### **AWS Authentication**
Credentials managed through standard AWS credential chain:

```bash
# Use specific AWS profile
aws-rstudio launch --profile research

# Custom region
aws-rstudio launch --region eu-west-1

# Custom idle timeout
aws-rstudio launch --idle-timeout 8h
```

### **Advanced Options**

```bash
# Combine all options
aws-rstudio launch \
  --connection session-manager \
  --subnet-type private \
  --create-nat-gateway \
  --env bioconductor \
  --instance-type t4g.xlarge \
  --idle-timeout 8h \
  --profile production \
  --region us-east-1
```

## 📋 Requirements

### **System Requirements**
- **Go 1.22+** for building from source
- **AWS CLI** configured with appropriate credentials
- **Operating System**: Linux, macOS, or Windows

### **AWS Permissions**
See the main [README](../../README.md#aws-permissions) for required AWS permissions.

## 📖 Documentation

### **Guides**
- **[Session Manager Setup](../jupyter/docs/SESSION_MANAGER_SETUP.md)** - Complete setup guide for AWS Session Manager
- **[Private Subnet Guide](../jupyter/docs/PRIVATE_SUBNET_GUIDE.md)** - Best practices for private subnet deployments
- **[Troubleshooting](../jupyter/docs/TROUBLESHOOTING.md)** - Common issues and solutions
- **[AWS Authentication](../jupyter/docs/AWS_AUTHENTICATION.md)** - Complete authentication setup guide

### **Reference**
- **[Main README](../../README.md)** - Overview of the AWS IDE project
- **[Roadmap](../../ROADMAP.md)** - Detailed feature planning
- **[Contributing](../../CONTRIBUTING.md)** - How to contribute

## Development Status

**Current Version**: v0.5.0 (Feature Parity - In Progress)

aws-rstudio is rapidly approaching feature parity with [aws-jupyter](../jupyter/). Most core features are implemented and tested.

### ✅ Working Features
- **Full CLI command set**: All 10 commands implemented
  - Launch, connect, stop, terminate, status, list
  - Environment management (env list, env show)
  - Key management (key list, key show, key validate)
- **Environment system**: 4 R-specific environments (minimal, tidyverse, bioconductor, shiny)
- **Connection methods**: SSH and Session Manager fully supported
- **Networking**: Public/private subnet support with NAT Gateway
- **Security**: IAM roles, security groups, automatic firewall rules
- **Cost optimization**: Idle detection and auto-stop with configurable timeouts
- **Test coverage**: 27 unit tests across env, generate, list, and launch commands

### 🚧 In Development (v0.5.0 - Q1 2025)
- Custom environment generation command
- RStudio-specific user data script enhancements
- Additional R package management features
- Documentation expansion

### 📋 Planned (v0.6.0+)
- Integration and E2E testing for RStudio-specific workflows
- Enhanced idle detection for R processes
- Shiny app port configuration
- R environment templates library

See the [ROADMAP](../../ROADMAP.md) for detailed planning.

## Development

```bash
# Clone and build
git clone https://github.com/scttfrdmn/aws-ide
cd aws-ide/apps/rstudio
go build ./cmd/aws-rstudio

# Run tests
go test ./...

# Build for release
cd ../..
goreleaser build --snapshot --rm-dist
```

## Shared Infrastructure

aws-rstudio shares infrastructure with other AWS IDE tools:

- **AWS SDK integrations**: `../../pkg/aws/`
- **Configuration management**: `../../pkg/config/`
- **CLI utilities**: `../../pkg/cli/`

This allows for consistent behavior across all IDE types and reduces code duplication.

## Contributing

We welcome contributions! Please see the main [Contributing Guide](../../CONTRIBUTING.md) for details.

**Priority areas for aws-rstudio:**
- R-specific environment templates
- RStudio Server configuration
- R package management
- Testing and documentation

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](../../LICENSE) file for details.

## Support

- **Documentation**: See guides in `../jupyter/docs/`
- **Issues**: [GitHub Issues](https://github.com/scttfrdmn/aws-ide/issues)
- **Discussions**: [GitHub Discussions](https://github.com/scttfrdmn/aws-ide/discussions)

## Acknowledgments

Built with:
- [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2)
- [Cobra CLI framework](https://github.com/spf13/cobra)
- Shared infrastructure from the AWS IDE project
