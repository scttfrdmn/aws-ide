# AWS IDE

A powerful CLI toolkit for launching secure cloud-based development environments on AWS EC2 with professional-grade networking and security features.

## Overview

AWS IDE is a monorepo containing multiple CLI tools for managing different cloud-based IDEs on AWS:

- **[aws-jupyter](apps/jupyter/)** - Launch and manage Jupyter Lab instances
- **[aws-rstudio](apps/rstudio/)** - Launch and manage RStudio Server instances
- **[aws-vscode](apps/vscode/)** - Launch and manage VSCode Server (code-server) instances

## Key Features

### 🔒 Security & Access Control
- **Session Manager**: Secure access without SSH keys or bastion hosts
- **SSM-based Readiness Polling**: Health checks from inside instances without exposed ports
- **Traditional SSH**: Full SSH key management with economical reuse strategy
- **Private Subnets**: Enterprise-grade isolation with optional NAT Gateway
- **Smart Security Groups**: Automatic firewall rules with IP restrictions

### 🏗️ Infrastructure Management
- **Zero Infrastructure**: Deploy from your laptop with full AWS integration
- **Graviton Optimized**: ARM64 instances for best price/performance ratio
- **Advanced Networking**: Public/private subnet support with NAT Gateway creation
- **Resource Reuse**: Smart reuse of existing infrastructure to minimize costs

### ⚙️ Environment System
- **Built-in Environments**: Pre-configured environments for different use cases
- **Auto-Generation**: Create custom environments from your local setup
- **YAML Configuration**: Simple, version-controlled environment definitions
- **Package Management**: Automatic handling of system packages and dependencies

### 💰 Cost Optimization
- **Automatic Idle Detection**: Multi-signal monitoring (kernels, CPU, processes)
- **Auto-Stop**: Configurable idle timeout to prevent runaway costs
- **Flexible Timeouts**: Set custom idle timeouts (e.g., `--idle-timeout 30m`, `2h`, `8h`)
- **Smart Monitoring**: Detects active sessions, CPU usage, and running computations

## Quick Start

### Install via Package Managers

**Homebrew (macOS/Linux)**
```bash
brew tap scttfrdmn/tap
brew install aws-jupyter aws-rstudio aws-vscode
```

**Scoop (Windows)**
```bash
scoop bucket add scttfrdmn https://github.com/scttfrdmn/scoop-bucket
scoop install aws-jupyter aws-rstudio aws-vscode
```

**APT (Debian/Ubuntu)**
```bash
# Download and install deb packages
wget https://github.com/scttfrdmn/aws-ide/releases/latest/download/aws-jupyter_0.5.1_linux_amd64.deb
wget https://github.com/scttfrdmn/aws-ide/releases/latest/download/aws-rstudio_0.5.1_linux_amd64.deb
wget https://github.com/scttfrdmn/aws-ide/releases/latest/download/aws-vscode_0.5.1_linux_amd64.deb

sudo dpkg -i aws-jupyter_0.5.1_linux_amd64.deb
sudo dpkg -i aws-rstudio_0.5.1_linux_amd64.deb
sudo dpkg -i aws-vscode_0.5.1_linux_amd64.deb
```

**YUM/DNF (RedHat/Fedora/Amazon Linux)**
```bash
# Download and install rpm packages
wget https://github.com/scttfrdmn/aws-ide/releases/latest/download/aws-jupyter_0.5.1_linux_amd64.rpm
wget https://github.com/scttfrdmn/aws-ide/releases/latest/download/aws-rstudio_0.5.1_linux_amd64.rpm
wget https://github.com/scttfrdmn/aws-ide/releases/latest/download/aws-vscode_0.5.1_linux_amd64.rpm

sudo rpm -i aws-jupyter_0.5.1_linux_amd64.rpm
sudo rpm -i aws-rstudio_0.5.1_linux_amd64.rpm
sudo rpm -i aws-vscode_0.5.1_linux_amd64.rpm
```

### Install from Source

```bash
# Clone the repository
git clone https://github.com/scttfrdmn/aws-ide
cd aws-ide

# Build Jupyter launcher
cd apps/jupyter
go build -o aws-jupyter ./cmd/aws-jupyter
sudo mv aws-jupyter /usr/local/bin/

# Build RStudio launcher
cd ../rstudio
go build -o aws-rstudio ./cmd/aws-rstudio
sudo mv aws-rstudio /usr/local/bin/

# Build VSCode launcher
cd ../vscode
go build -o aws-vscode ./cmd/aws-vscode
sudo mv aws-vscode /usr/local/bin/
```

### Launch Your First Instance

```bash
# Launch Jupyter Lab with all defaults
aws-jupyter launch

# Launch RStudio Server
aws-rstudio launch

# Launch VSCode Server
aws-vscode launch

# Launch with Session Manager (no SSH)
aws-jupyter launch --connection session-manager

# Launch with custom timeout
aws-jupyter launch --idle-timeout 2h
```

## How It Works: SSM-Based Readiness Polling

AWS IDE uses AWS Systems Manager (SSM) for secure, agentless service health checks during instance launch:

**The Problem**: Traditional health checks require exposing service ports through security groups, creating security risks and complexity.

**The Solution**: SSM-based polling checks service health from **inside** the instance using AWS Systems Manager:

1. Launch instance with IAM instance profile (SSM access included)
2. Wait for SSM agent to come online (typically 5-10 seconds)
3. Execute `curl localhost:<port>` commands via SSM to check service readiness
4. Display real-time progress with cloud-init logs streamed via SSH
5. Report when service is ready (typically 2-3 minutes total)

**Benefits**:
- Works regardless of security group configuration
- No need to expose service ports externally for testing
- More secure launch process with reduced attack surface
- Unified health checking across all IDE types (VSCode, Jupyter, RStudio)
- Real-time progress visibility with concurrent progress streaming

All three apps (aws-jupyter, aws-rstudio, aws-vscode) use this approach by default.

## Applications

### [AWS Jupyter](apps/jupyter/)

Full-featured Jupyter Lab launcher with:
- 6 built-in environments (data-science, ml-pytorch, deep-learning, etc.)
- Custom environment generation
- Complete lifecycle management
- Auto-stop and idle detection

[Read the full Jupyter documentation →](apps/jupyter/README.md)

### [AWS RStudio](apps/rstudio/)

RStudio Server launcher with:
- R-optimized environments
- Tidyverse and data science packages
- Session Manager support
- Shared infrastructure with Jupyter

[Read the full RStudio documentation →](apps/rstudio/README.md)

### [AWS VSCode](apps/vscode/)

VSCode Server (code-server) launcher with:
- 4 built-in environments (web-dev, python-dev, go-dev, fullstack)
- Automatic extension installation
- Full browser-based VSCode experience
- Language runtime setup (Node.js, Python, Go)

[Read the full VSCode documentation →](apps/vscode/README.md)

## Architecture

This project is organized as a Go workspace with shared infrastructure:

```
aws-ide/
├── pkg/                    # Shared library
│   ├── aws/               # AWS SDK integrations (EC2, IAM, networking)
│   ├── cli/               # Common CLI utilities
│   └── config/            # Configuration and state management
├── apps/
│   ├── jupyter/           # Jupyter Lab launcher
│   │   ├── cmd/           # Entry point
│   │   ├── internal/      # App-specific code
│   │   ├── docs/          # Jupyter-specific documentation
│   │   └── environments/  # Built-in Jupyter environments
│   ├── rstudio/           # RStudio Server launcher
│   │   ├── cmd/           # Entry point
│   │   ├── internal/      # App-specific code
│   │   └── environments/  # Built-in RStudio environments
│   └── vscode/            # VSCode Server launcher
│       ├── cmd/           # Entry point
│       ├── internal/      # App-specific code
│       └── environments/  # Built-in VSCode environments
└── go.work                # Go workspace configuration
```

## AWS Authentication

Before using any AWS IDE tools, configure AWS credentials:

```bash
# Option 1: AWS CLI (Recommended)
aws configure

# Option 2: Environment variables
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_DEFAULT_REGION="us-east-1"

# Option 3: AWS profiles
aws configure --profile myprofile
aws-jupyter launch --profile myprofile
```

See the [Jupyter AWS Authentication Guide](apps/jupyter/docs/AWS_AUTHENTICATION.md) for detailed setup instructions.

## Versioning

AWS IDE uses **unified versioning** across all apps in the monorepo. All three tools share the same version number and are released together.

- **Current Release**: v0.5.1
- **aws-jupyter**: v0.5.1 (production)
- **aws-rstudio**: v0.5.1 (production)
- **aws-vscode**: v0.5.1 (beta)

Releases use semantic versioning with Git tags: `v0.5.1`, `v0.6.0`, etc.

See [VERSIONING.md](VERSIONING.md) for detailed versioning strategy and release process.

## Development

### Prerequisites

- Go 1.22 or later
- AWS CLI configured with appropriate credentials
- Git

### Building

```bash
# Build all applications
cd apps/jupyter && go build ./cmd/aws-jupyter
cd ../rstudio && go build ./cmd/aws-rstudio
cd ../vscode && go build ./cmd/aws-vscode

# Run tests
cd ../../pkg && go test ./...
cd ../apps/jupyter && go test ./...
cd ../apps/rstudio && go test ./...
cd ../apps/vscode && go test ./...
```

### Running Tests

```bash
# Test shared library
cd pkg
go test -v ./...

# Test Jupyter
cd apps/jupyter
go test -v ./...

# Test RStudio
cd apps/rstudio
go test -v ./...

# Test VSCode
cd ../vscode
go test -v ./...
```

### Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## Roadmap

### ✅ Completed
- **v0.1.0-v0.4.0**: Core Jupyter features, Session Manager, cost optimization
- **Monorepo Migration**: Shared infrastructure for multiple IDE types

### 🚀 In Progress
- **aws-vscode**: VSCode Server implementation (alpha)

### 📋 Planned
- **v0.6.0** (Q1 2025): Complete aws-vscode launch command, UX enhancements
- **v0.7.0** (Q1 2025): Integration testing (40%+ coverage)
- **v0.8.0-v1.0.0**: Multi-instance ops, backups, enterprise features
- **RStudio Feature Parity**: Port all Jupyter features to RStudio
- **Additional IDEs**: Theia, Apache Zeppelin, Streamlit, etc.

See [ROADMAP.md](ROADMAP.md) for detailed planning.

## AWS Permissions

Your AWS credentials need these permissions:

### Core Permissions (All Tools)
- `ec2:DescribeInstances`, `ec2:RunInstances`, `ec2:TerminateInstances`
- `ec2:DescribeImages`, `ec2:DescribeInstanceTypes`
- `ec2:DescribeVpcs`, `ec2:DescribeSubnets`
- `ec2:CreateTags`, `ec2:DescribeTags`

### SSH Connection Method
- `ec2:CreateKeyPair`, `ec2:DescribeKeyPairs`, `ec2:DeleteKeyPair`
- `ec2:CreateSecurityGroup`, `ec2:DescribeSecurityGroups`
- `ec2:AuthorizeSecurityGroupIngress`

### Session Manager Connection
- `iam:CreateRole`, `iam:GetRole`, `iam:AttachRolePolicy`
- `iam:CreateInstanceProfile`, `iam:AddRoleToInstanceProfile`
- `ssm:StartSession`

### Private Subnet with NAT Gateway
- `ec2:CreateNatGateway`, `ec2:DescribeNatGateways`
- `ec2:AllocateAddress`, `ec2:DescribeAddresses`
- `ec2:CreateRoute`, `ec2:DescribeRouteTables`

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: See app-specific READMEs and docs directories
- **Issues**: [GitHub Issues](https://github.com/scttfrdmn/aws-ide/issues)
- **Discussions**: [GitHub Discussions](https://github.com/scttfrdmn/aws-ide/discussions)

## Acknowledgments

Built with:
- [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2)
- [Cobra CLI framework](https://github.com/spf13/cobra)
- [GoReleaser](https://goreleaser.com/)
