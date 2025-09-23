# aws-jupyter

[![Go Report Card](https://goreportcard.com/badge/github.com/scttfrdmn/aws-jupyter)](https://goreportcard.com/report/github.com/scttfrdmn/aws-jupyter)
[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/scttfrdmn/aws-jupyter)](https://github.com/scttfrdmn/aws-jupyter/releases)

> **⚠️ UNDER ACTIVE DEVELOPMENT**: This project is currently in active development. Core functionality including AWS instance launching, SSH tunneling, and state management are being implemented. See the [project roadmap](#roadmap) and [contributing guide](CONTRIBUTING.md) to get involved.

A CLI tool for quickly launching Jupyter Lab instances on AWS EC2 Graviton processors with automatic SSH tunneling and idle detection.

## Features

- **Zero infrastructure**: Run entirely from your laptop
- **Graviton optimized**: Targets ARM64 instances for best price/performance
- **Simple environments**: YAML-based environment configurations
- **Auto-shutdown**: Configurable idle detection and hibernation support
- **SSH tunneling**: Automatic port forwarding to localhost
- **State management**: Track and manage multiple instances

## Installation

```bash
go install github.com/scttfrdmn/aws-jupyter@latest
```

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

```bash
# Launch with default data science environment
aws-jupyter launch

# Launch with specific environment and instance type
aws-jupyter launch --env ml-pytorch --instance-type m7g.large

# Generate environment from your local setup
aws-jupyter generate --name my-env --source ./my-project

# List running instances
aws-jupyter list

# Connect to existing instance
aws-jupyter connect i-0abc123def

# Stop instance (with hibernation)
aws-jupyter stop i-0abc123def --hibernate

# Terminate instance
aws-jupyter terminate i-0abc123def
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

## Configuration

AWS credentials are managed through standard AWS credential chain (profiles, environment variables, IAM roles).

```bash
# Use specific AWS profile
aws-jupyter launch --profile research

# Custom idle timeout
aws-jupyter launch --idle-timeout 8h
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

## Requirements

- Go 1.21+
- AWS CLI configured with appropriate permissions
- EC2, VPC permissions for launching instances

## Roadmap

### ✅ **Phase 1: Foundation** (Complete)
- [x] CLI framework with Cobra commands
- [x] Environment configuration system (6 built-in environments)
- [x] AWS client integration and authentication
- [x] Environment generation from local setups
- [x] Comprehensive test coverage (74%+)
- [x] Dry-run functionality

### 🚧 **Phase 2: AWS Resource Management** (In Progress)
- [ ] SSH key pair management
- [ ] Security group setup (SSH + Jupyter ports)
- [ ] User data script generation for environment setup

### 📋 **Phase 3: Instance Lifecycle** (Planned)
- [ ] EC2 instance launching and provisioning
- [ ] Instance state tracking and persistence
- [ ] Stop/start/terminate functionality

### 🎯 **Phase 4: Connectivity & UX** (Planned)
- [ ] SSH tunnel management (local port forwarding)
- [ ] Connect command for existing instances
- [ ] Real-time status monitoring
- [ ] Idle detection and auto-shutdown

See our [Contributing Guide](CONTRIBUTING.md) to help implement these features!

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