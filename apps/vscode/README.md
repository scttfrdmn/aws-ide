# aws-vscode

Launch VSCode Server (code-server) instances on AWS EC2 with professional-grade security and networking.

## Overview

`aws-vscode` enables you to quickly spin up cloud-based VSCode environments on AWS Graviton processors, providing a full VSCode experience in your browser with automatic configuration of development tools, language runtimes, and extensions.

## Features

- **Full VSCode Experience**: Browser-based VSCode Server with all your favorite extensions
- **Multiple Environments**: Pre-configured presets for web development, Python, Go, and fullstack projects
- **AWS Graviton**: Cost-effective ARM64 instances (m7g family)
- **Flexible Connectivity**: SSH tunnels or AWS Session Manager port forwarding
- **Smart Networking**: Public or private subnets with automatic NAT Gateway creation
- **Security First**: Proper security groups, key management, and IAM instance profiles
- **Auto-Stop**: Configurable idle timeouts to minimize costs
- **Ubuntu 24.04 LTS**: Modern, long-term supported base OS

## Quick Start

### Installation

#### Homebrew (macOS/Linux)

```bash
brew install scttfrdmn/tap/aws-vscode
```

#### From Source

```bash
git clone https://github.com/scttfrdmn/aws-ide
cd aws-ide/apps/vscode
go build ./cmd/aws-vscode
```

### Prerequisites

1. **AWS CLI** configured with credentials:
   ```bash
   aws configure
   ```

2. **Session Manager Plugin** (for private subnets):
   ```bash
   # macOS
   brew install --cask session-manager-plugin

   # Linux
   curl "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/ubuntu_64bit/session-manager-plugin.deb" -o "session-manager-plugin.deb"
   sudo dpkg -i session-manager-plugin.deb
   ```

### Launch Your First Instance

```bash
# Launch with default web-dev environment
aws-vscode launch

# Launch with specific environment
aws-vscode launch --environment python-dev

# Launch in specific region
aws-vscode launch --region us-west-2 --profile myprofile
```

### Connect to Your Instance

```bash
# Auto-connect to your instance
aws-vscode connect

# Or specify instance ID
aws-vscode connect i-1234567890abcdef0
```

Then open http://localhost:8080 in your browser!

### Get Password

The VSCode Server password is randomly generated on first launch. To retrieve it:

```bash
# SSH into the instance
aws-vscode connect i-1234567890abcdef0

# Then on the instance
cat ~/.config/code-server/config.yaml
```

The default password is `vscode2024` (you can change it by editing the config file).

## Environments

### web-dev (Default)
Perfect for frontend development with React, Vue, Svelte, etc.

**Includes:**
- Node.js 20 LTS
- TypeScript, ESLint, Prettier
- VSCode Extensions: ESLint, Prettier, Tailwind CSS, Auto Rename Tag, Live Server

**Instance:** m7g.medium, 30GB EBS

**Estimated Cost:** ~$0.03/hour + storage

### python-dev
Full Python development environment with data science tools.

**Includes:**
- Python 3 with pip
- Black, Flake8, Pylint, MyPy, pytest
- Pandas, NumPy, Matplotlib
- VSCode Extensions: Python, Pylance, Black Formatter, Jupyter

**Instance:** m7g.medium, 40GB EBS

**Estimated Cost:** ~$0.03/hour + storage

### go-dev
Clean Go development setup.

**Includes:**
- Go 1.22
- VSCode Extensions: Go, Go Nightly

**Instance:** m7g.medium, 30GB EBS

**Estimated Cost:** ~$0.03/hour + storage

### fullstack
Combined environment for fullstack development.

**Includes:**
- Node.js 20 LTS + Python 3 + PM2
- TypeScript, ESLint, Prettier
- Django, FastAPI, SQLAlchemy, Celery
- PostgreSQL client, Redis tools
- VSCode Extensions: Python, ESLint, Prettier, Docker, Tailwind CSS

**Instance:** m7g.large, 50GB EBS

**Estimated Cost:** ~$0.06/hour + storage

## Commands

### launch
```bash
aws-vscode launch [flags]
```

Launch a new VSCode Server instance.

**Flags:**
- `-e, --environment`: Environment preset (web-dev, python-dev, go-dev, fullstack)
- `-p, --profile`: AWS profile to use
- `-r, --region`: AWS region

**Note:** The full launch implementation is in progress. Currently shows a placeholder message.

### list
```bash
aws-vscode list
```

List all running VSCode Server instances with status.

### connect
```bash
aws-vscode connect [INSTANCE_ID]
```

Setup port forwarding to access VSCode Server on http://localhost:8080.

### status
```bash
aws-vscode status INSTANCE_ID
```

Show detailed instance information including uptime, IPs, and tunnel status.

### start/stop
```bash
aws-vscode stop INSTANCE_ID
aws-vscode start INSTANCE_ID
```

Stop or start an instance. Data is preserved on EBS volume.

### terminate
```bash
aws-vscode terminate INSTANCE_ID
```

Permanently delete an instance. **This action cannot be undone.**

### env
```bash
aws-vscode env list
aws-vscode env validate ENV_NAME
```

Manage and validate environment configurations.

### key
```bash
aws-vscode key list
aws-vscode key show [KEY_NAME]
aws-vscode key validate
aws-vscode key cleanup
```

Manage SSH key pairs.

## Architecture

### Networking
- **Public Subnet**: Direct internet access via Internet Gateway
- **Private Subnet**: Internet access via NAT Gateway (more secure)
- Security groups only allow your IP address

### Compute
- **Default**: m7g.medium (2 vCPU, 8GB RAM, Graviton3)
- **Fullstack**: m7g.large (2 vCPU, 16GB RAM, Graviton3)
- Ubuntu 24.04 Noble LTS (ARM64)

### Storage
- EBS gp3 volumes (30-50GB depending on environment)
- Persists across stop/start cycles

### Security
- IAM instance profile for Session Manager access
- Security groups restrict access to your IP
- SSH keys stored in ~/.aws-vscode/keys/
- Password-protected VSCode Server

## Cost Optimization

### Auto-Stop Feature
Instances automatically stop after being idle to minimize costs:

```bash
# Default: 4 hours idle timeout
aws-vscode launch --idle-timeout 4h

# Disable auto-stop
aws-vscode launch --idle-timeout 0
```

### Manual Management
```bash
# Stop when done working
aws-vscode stop INSTANCE_ID

# Start when needed again
aws-vscode start INSTANCE_ID

# Terminate when completely done
aws-vscode terminate INSTANCE_ID
```

### Estimated Monthly Costs
**Scenario**: 8 hours/day, 20 days/month, m7g.medium

| Component | Cost |
|-----------|------|
| EC2 (160 hours/month) | ~$4.80 |
| EBS (30GB, 24/7) | ~$2.40 |
| Data Transfer | ~$0.50 |
| **Total** | **~$7.70/month** |

**With auto-stop**: Costs can be reduced by 60-70% depending on usage patterns.

## Customization

### Create Custom Environment

1. Copy an existing environment file:
   ```bash
   cp environments/web-dev.yaml environments/my-custom.yaml
   ```

2. Edit the YAML file:
   ```yaml
   name: "My Custom Environment"
   instance_type: "m7g.medium"
   ami_base: "ubuntu24-arm64"
   ebs_volume_size: 40
   packages:
     - git
     - curl
   environment_vars:
     NODEJS_VERSION: "20"
     VSCODE_EXTENSIONS: "your.extension,another.extension"
   ```

3. Launch with your environment:
   ```bash
   aws-vscode launch --environment my-custom
   ```

### Add VSCode Extensions

Extensions are specified in the `VSCODE_EXTENSIONS` environment variable as a comma-separated list:

```yaml
environment_vars:
  VSCODE_EXTENSIONS: "dbaeumer.vscode-eslint,esbenp.prettier-vscode,ms-python.python"
```

## Troubleshooting

### Can't connect to instance
```bash
# Check instance is running
aws-vscode status INSTANCE_ID

# Check security group allows your IP
# Security groups are updated when you launch, but your IP may change
```

### VSCode Server not starting
```bash
# SSH to instance and check logs
aws ssm start-session --target INSTANCE_ID

# Check service status
sudo systemctl status code-server

# View logs
sudo journalctl -u code-server -f
```

### Port 8080 already in use
```bash
# Use different local port
aws-vscode connect INSTANCE_ID --port 8090
```

## Comparison with Other Solutions

| Feature | aws-vscode | GitHub Codespaces | VS Code Remote SSH | Gitpod |
|---------|------------|-------------------|-------------------|--------|
| **Cost** | ~$7/month | $0.18/hour | Local + SSH setup | ~$9/month |
| **Control** | Full AWS control | GitHub managed | Full control | Gitpod managed |
| **Setup** | 1 command | Immediate | Manual | Git-based |
| **Offline** | No | No | Yes | No |
| **Customization** | Full | Limited | Full | Dockerfiles |

## Development Status

**Current Version:** 0.1.0 (Alpha)

### ✅ Completed
- Core CLI structure
- All subcommands (list, connect, stop, start, terminate, status, env, key)
- User-data scripts for code-server setup
- 4 environment presets
- Build system integrated with monorepo

### 🚧 In Progress
- Full launch command implementation (currently placeholder)
- Integration testing with AWS
- Documentation refinement

### 📋 Planned
- HTTPS support with Let's Encrypt
- GitHub OAuth authentication
- Workspace templates
- Custom domain support
- Docker container support

## Contributing

This is part of the [aws-ide monorepo](https://github.com/scttfrdmn/aws-ide). See the main README for contribution guidelines.

## License

Apache 2.0

## Related Projects

- [aws-jupyter](../jupyter) - Jupyter Lab on AWS Graviton
- [aws-rstudio](../rstudio) - RStudio Server on AWS Graviton
- [code-server](https://github.com/coder/code-server) - VS Code in the browser

## Support

- Issues: https://github.com/scttfrdmn/aws-ide/issues
- Discussions: https://github.com/scttfrdmn/aws-ide/discussions

---

**Built with ❤️ for developers who need flexible, cost-effective cloud development environments.**
