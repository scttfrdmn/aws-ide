# aws-jupyter

[![Go Report Card](https://goreportcard.com/badge/github.com/scttfrdmn/aws-jupyter)](https://goreportcard.com/report/github.com/scttfrdmn/aws-jupyter)
[![Go Version](https://img.shields.io/badge/go-1.22+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/scttfrdmn/aws-jupyter)](https://github.com/scttfrdmn/aws-jupyter/releases)

A powerful CLI tool for launching secure Jupyter Lab instances on AWS EC2 Graviton processors with professional-grade networking and security features.

**Full lifecycle management** with 10 commands for launching, connecting, stopping, and terminating instances across public and private subnets with Session Manager or SSH access.

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
- **Built-in Environments**: 6 pre-configured environments for different use cases
- **Auto-Generation**: Create custom environments from your local Python setup
- **YAML Configuration**: Simple, version-controlled environment definitions
- **Package Management**: Automatic handling of system packages, pip, and Jupyter extensions

### **🛠️ Developer Experience**
- **Dry Run Mode**: Preview all changes before resource creation
- **Cost Awareness**: Clear warnings about additional charges (NAT Gateway, etc.)
- **Comprehensive CLI**: Full lifecycle management with intuitive commands
- **State Tracking**: Persistent local state for managing multiple instances

## Installation

### Homebrew (macOS and Linux)

```bash
brew tap scttfrdmn/tap
brew install aws-jupyter
```

### Go Install

```bash
go install github.com/scttfrdmn/aws-jupyter@latest
```

### Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/scttfrdmn/aws-jupyter/releases).

## AWS Authentication

Before using aws-jupyter, you need to configure AWS credentials. The tool supports all standard AWS authentication methods:

📋 **[Complete AWS Authentication Guide →](docs/AWS_AUTHENTICATION.md)**

**Quick Setup Options:**

- **AWS Profiles** (Recommended): `aws configure --profile myprofile`
- **Environment Variables**: Set `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`
- **AWS SSO**: `aws sso login --profile mysso`
- **IAM Roles**: Automatic when running on EC2/ECS/Lambda

```bash
# Verify your AWS access
aws sts get-caller-identity --profile myprofile

# Use with aws-jupyter
aws-jupyter launch --profile myprofile --region us-west-2
```

## Quick Start

### **🔐 Secure Launch (Session Manager - Recommended)**
```bash
# Most secure: Private subnet with Session Manager
aws-jupyter launch --connection session-manager --subnet-type private --create-nat-gateway

# Secure and cost-effective: Private subnet without internet
aws-jupyter launch --connection session-manager --subnet-type private

# Session Manager with public subnet (no SSH exposure)
aws-jupyter launch --connection session-manager
```

### **🔑 Traditional SSH Launch**
```bash
# Standard SSH with public subnet
aws-jupyter launch --connection ssh

# SSH with custom environment and instance type
aws-jupyter launch --env ml-pytorch --instance-type m7g.large --connection ssh
```

### **🛠️ Instance & Resource Management**
```bash
# Instance lifecycle
aws-jupyter list                        # Show all instances with status
aws-jupyter status i-0abc123def         # Detailed instance information
aws-jupyter connect i-0abc123def        # Connect to existing instance
aws-jupyter stop i-0abc123def           # Stop instance (preserves EBS)
aws-jupyter terminate i-0abc123def      # Terminate instance (cleanup)

# SSH key management
aws-jupyter key list                    # View local and AWS key pairs
aws-jupyter key show                    # Show default key details
aws-jupyter key validate                # Check key file permissions
aws-jupyter key cleanup --dry-run       # Preview orphaned key cleanup

# Environment generation
aws-jupyter generate --name my-env --source ./my-project
```

### **📋 Preview Changes (Dry Run)**
```bash
# Always preview before launching
aws-jupyter launch --dry-run --connection session-manager --subnet-type private --create-nat-gateway
```

## Environments

Built-in environments:
- `data-science`: General data analysis with pandas, numpy, matplotlib, scikit-learn
- `ml-pytorch`: Machine learning with PyTorch, transformers, datasets
- `deep-learning`: Advanced ML with PyTorch, TensorFlow, MLflow, Optuna
- `r-stats`: R statistical computing with Jupyter R kernel, tidyverse
- `computational-bio`: Bioinformatics with biopython, samtools, bedtools
- `minimal`: Basic Python environment with just essentials

## Generating Custom Environments

Create environments from your local setup:

```bash
# Analyze current directory and generate config
aws-jupyter generate --name my-project

# Analyze specific directory with notebooks
aws-jupyter generate --source ./research --name research-env

# Generate from requirements.txt
aws-jupyter generate --source requirements.txt --name prod-env

# Generate without scanning notebooks
aws-jupyter generate --name simple-env --scan-notebooks=false
```

The generate command will:
- Parse `requirements.txt` files
- Analyze conda environments (if available)
- Scan current pip environment
- Extract imports from `.ipynb` files
- Suggest appropriate instance types and storage
- Create a complete environment YAML file

### Example Generated Config

```yaml
name: "My Research Project"
instance_type: "m7g.large"
ami_base: "ubuntu22-arm64"
ebs_volume_size: 30
packages:
  - python3-pip
  - python3-dev
  - jupyter
  - git
  - htop
  - awscli
pip_packages:
  - jupyterlab
  - notebook
  - pandas
  - numpy
  - matplotlib
  - torch
  - transformers
  - scikit-learn
jupyter_extensions:
  - jupyterlab
environment_vars:
  PYTHONPATH: "/home/ubuntu/notebooks"
```

## 🔗 Connection Methods

### **Session Manager (Recommended)**
- **No SSH keys required** - eliminates key management complexity
- **Enhanced security** - access through AWS SSM, no direct internet exposure
- **Audit logging** - all sessions logged in CloudTrail
- **Works anywhere** - no bastion hosts or VPN required

📋 **[Complete Session Manager Setup Guide →](docs/SESSION_MANAGER_SETUP.md)**

```bash
# Launch with Session Manager
aws-jupyter launch --connection session-manager

# Connect to running instance
aws-jupyter connect i-0abc123def
# or use AWS CLI directly:
aws ssm start-session --target i-0abc123def --profile myprofile
```

**Prerequisites**: Requires AWS CLI and Session Manager plugin installed. See setup guide for details.

### **Traditional SSH**
- **Full SSH access** - direct SSH connection with automatic key management
- **Economical key reuse** - smart naming strategy (aws-jupyter-{region})
- **Secure local storage** - private keys stored with 600 permissions
- **IP restrictions** - security groups restrict access to your current IP

```bash
# Launch with SSH
aws-jupyter launch --connection ssh

# Connect directly
ssh -i ~/.aws-jupyter/keys/aws-jupyter-us-west-2.pem ec2-user@1.2.3.4
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

📋 **[Private Subnet Best Practices Guide →](docs/PRIVATE_SUBNET_GUIDE.md)**

```bash
# Private subnet with internet access (recommended for production)
aws-jupyter launch --subnet-type private --create-nat-gateway

# Private subnet without internet (cost-effective but limited functionality)
aws-jupyter launch --subnet-type private

# Cost breakdown displayed during dry-run
aws-jupyter launch --dry-run --subnet-type private --create-nat-gateway
```

## ⚙️ Configuration

### **AWS Authentication**
Credentials managed through standard AWS credential chain:

```bash
# Use specific AWS profile
aws-jupyter launch --profile research

# Custom region
aws-jupyter launch --region eu-west-1

# Custom idle timeout
aws-jupyter launch --idle-timeout 8h
```

### **Advanced Options**
```bash
# Combine all options
aws-jupyter launch \
  --connection session-manager \
  --subnet-type private \
  --create-nat-gateway \
  --env deep-learning \
  --instance-type m7g.xlarge \
  --profile production \
  --region us-east-1
```

## Custom Environments

Create custom environments in `~/.aws-jupyter/environments/`:

```yaml
name: "My Custom Environment"
instance_type: "m7g.medium"
ami_base: "ubuntu22-arm64"
ebs_volume_size: 30
packages:
  - python3-pip
  - custom-package
pip_packages:
  - special-library==1.0.0
jupyter_extensions:
  - jupyterlab
```

## 📋 Requirements

### **System Requirements**
- **Go 1.22+** for building from source
- **AWS CLI** configured with appropriate credentials
- **Operating System**: Linux, macOS, or Windows

### **AWS Permissions**
Your AWS credentials need the following permissions:

#### **Core Permissions (All Methods)**
- `ec2:DescribeInstances`, `ec2:RunInstances`, `ec2:TerminateInstances`
- `ec2:DescribeImages`, `ec2:DescribeInstanceTypes`
- `ec2:DescribeVpcs`, `ec2:DescribeSubnets`
- `ec2:CreateTags`, `ec2:DescribeTags`

#### **SSH Connection Method**
- `ec2:CreateKeyPair`, `ec2:DescribeKeyPairs`, `ec2:DeleteKeyPair`
- `ec2:CreateSecurityGroup`, `ec2:DescribeSecurityGroups`
- `ec2:AuthorizeSecurityGroupIngress`, `ec2:RevokeSecurityGroupIngress`

#### **Session Manager Connection Method**
- `iam:CreateRole`, `iam:GetRole`, `iam:AttachRolePolicy`
- `iam:CreateInstanceProfile`, `iam:AddRoleToInstanceProfile`
- `ssm:StartSession` (for connecting to instances)

#### **Private Subnet with NAT Gateway**
- `ec2:CreateNatGateway`, `ec2:DescribeNatGateways`
- `ec2:AllocateAddress`, `ec2:DescribeAddresses`
- `ec2:CreateRoute`, `ec2:DescribeRouteTables`

**💡 Tip**: Use AWS managed policies like `PowerUserAccess` for development, or create custom policies for production.

## 📖 Documentation

### **Guides**
- **[Session Manager Setup](docs/SESSION_MANAGER_SETUP.md)** - Complete setup guide for AWS Session Manager
- **[Private Subnet Guide](docs/PRIVATE_SUBNET_GUIDE.md)** - Best practices for private subnet deployments
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and solutions
- **[Examples & Use Cases](docs/EXAMPLES.md)** - Real-world usage scenarios

### **Reference**
- **[AWS Authentication](docs/AWS_AUTHENTICATION.md)** - Complete authentication setup guide
- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute to the project
- **[Roadmap](ROADMAP.md)** - Detailed feature planning through v1.0.0

## 🗺️ Roadmap

See the complete [ROADMAP.md](ROADMAP.md) for detailed feature planning through v1.0.0.

### ✅ **v0.1.0 - Core Foundation** (Complete)
- [x] CLI framework with 10 complete commands
- [x] Environment system (6 built-in + custom generation)
- [x] AWS integration and authentication
- [x] Dry-run functionality for all operations
- [x] Comprehensive documentation

### ✅ **v0.2.0 - Production Ready** (Current Release)
- [x] **Full lifecycle management**: Launch, connect, stop, terminate commands
- [x] **SSH key management**: Complete CLI for key operations
- [x] **Session Manager**: Secure access without SSH keys
- [x] **Advanced networking**: Public/private subnets with NAT Gateway support
- [x] **Test coverage**: 18.7% overall (AWS: 3.8%, CLI: 27.8%, Config: 19.1%)
- [x] **Code quality**: A+ Go Report Card, comprehensive linting
- [x] **Documentation**: Complete guides for all features

### 📋 **v0.3.0 - Integration Testing** (Planned Q1 2025)
- [ ] Integration test infrastructure with localstack/moto
- [ ] 40%+ overall test coverage
- [ ] End-to-end testing for complete workflows
- [ ] GitHub Actions integration test workflow

### 🚀 **v0.4.0 - UX Enhancements** (Planned Q1 2025)
- [ ] Interactive launch wizard
- [ ] Color-coded output and progress bars
- [ ] Enhanced error messages with suggestions
- [ ] Configuration file support (~/.aws-jupyter/config.yaml)

### 📈 **Future Versions**
- **v0.5.0**: Cost tracking and optimization
- **v0.6.0**: Multi-instance batch operations
- **v0.7.0**: Backup and restore capabilities
- **v0.8.0**: Enterprise features (multi-account, RBAC)
- **v0.9.0**: Plugin system and IDE integrations
- **v1.0.0**: Production-grade stability (60%+ coverage, security audit)

**Want to contribute?** Check our [Contributing Guide](CONTRIBUTING.md) and [ROADMAP.md](ROADMAP.md) for detailed feature plans!

## Development

```bash
git clone https://github.com/scttfrdmn/aws-jupyter
cd aws-jupyter
go mod tidy

# Install pre-commit hooks (optional but recommended)
pip install pre-commit
pre-commit install

# Build locally
go build -o aws-jupyter cmd/aws-jupyter/main.go

# Or use GoReleaser for full release build
goreleaser build --snapshot --rm-dist
```

### Code Quality

This project maintains an A+ grade on [Go Report Card](https://goreportcard.com/report/github.com/scttfrdmn/aws-jupyter) through:

- Pre-commit hooks that enforce formatting, linting, and testing
- Static analysis with `golangci-lint`
- Automated testing and building with GoReleaser
- Semantic versioning (SemVer 2.0.0) for releases