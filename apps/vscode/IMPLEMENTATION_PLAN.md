# aws-vscode Implementation Plan

## Overview

Implement `aws-vscode` CLI tool for launching VSCode Server (code-server) instances on AWS EC2, following the established aws-ide monorepo pattern.

## Architecture

### One User : One Instance Model
- Each user launches their own dedicated EC2 instance
- VSCode Server runs as systemd service
- Access via web browser at `https://<instance-ip>:8080`
- Password authentication from config file
- Optional: Can be enhanced with custom domain + Let's Encrypt

## Technical Stack

### Code-Server
- **Project**: https://github.com/coder/code-server
- **License**: MIT
- **Installation**: Official `.deb` packages
- **Systemd**: Automatic service management
- **Config**: `~/.config/code-server/config.yaml`

### Default Configuration
- **Port**: 8080 (configurable)
- **Auth**: Password from config file
- **Bind address**: 0.0.0.0 (accessible externally)
- **Base OS**: Ubuntu 24.04 Noble ARM64

## Implementation Steps

### 1. Module Setup

```bash
cd apps/vscode
go mod init github.com/scttfrdmn/aws-ide/apps/vscode
go mod edit -replace github.com/scttfrdmn/aws-ide/pkg=../../pkg
```

### 2. Main Command (cmd/aws-vscode/main.go)

Similar to aws-jupyter and aws-rstudio:
- Root command with version info
- Subcommands: launch, list, status, connect, stop, terminate
- Key management commands
- Environment generation

### 3. User Data Script (internal/userdata/)

**Setup Steps:**
1. System updates
2. Install code-server via official .deb package
3. Configure code-server (bind address, port, password)
4. Enable systemd service
5. Install extensions (optional, per environment)
6. Setup workspace directories
7. Configure git, SSH keys
8. Install language runtimes (Node.js, Python, Go, etc.)

**Key Files:**
- `userdata.go` - Template generator
- `vscode_setup.sh` - Main setup script template

### 4. Environments

#### **web-dev** (Default)
```yaml
name: "web-dev"
ami_base: "ubuntu24-arm64"
instance_type: "m7g.medium"
ebs_volume_size: 30
packages:
  - curl
  - wget
  - git
  - build-essential
nodejs_version: "20"  # LTS
vscode_extensions:
  - dbaeumer.vscode-eslint
  - esbenp.prettier-vscode
  - bradlc.vscode-tailwindcss
  - ms-vscode.vscode-typescript-next
```

#### **python-dev**
```yaml
name: "python-dev"
ami_base: "ubuntu24-arm64"
instance_type: "m7g.medium"
ebs_volume_size: 40
packages:
  - python3-pip
  - python3-venv
  - python3-dev
pip_packages:
  - black
  - flake8
  - pylint
  - pytest
vscode_extensions:
  - ms-python.python
  - ms-python.vscode-pylance
  - ms-python.black-formatter
  - ms-toolsai.jupyter
```

#### **go-dev**
```yaml
name: "go-dev"
ami_base: "ubuntu24-arm64"
instance_type: "m7g.medium"
ebs_volume_size: 30
packages:
  - git
  - build-essential
go_version: "1.22"
vscode_extensions:
  - golang.go
  - golang.go-nightly
```

#### **fullstack**
```yaml
name: "fullstack"
ami_base: "ubuntu24-arm64"
instance_type: "m7g.large"  # More resources
ebs_volume_size: 50
packages:
  - python3-pip
  - python3-venv
  - build-essential
  - postgresql-client
nodejs_version: "20"
pip_packages:
  - django
  - fastapi
  - sqlalchemy
vscode_extensions:
  - ms-python.python
  - dbaeumer.vscode-eslint
  - bradlc.vscode-tailwindcss
  - ms-azuretools.vscode-docker
```

### 5. Security Considerations

#### Network Security:
- Security group allows port 8080 from user's IP
- Optional: Session Manager for SSH access
- Optional: HTTPS with Let's Encrypt (future)

#### Authentication:
- Password-based by default
- Password stored in `~/.config/code-server/config.yaml`
- Display password to user after launch
- Optional: GitHub OAuth (future enhancement)

#### Data Persistence:
- User's workspace on EBS volume
- Survives stop/start cycles
- Backed up with EBS snapshots (optional)

### 6. User Experience

#### Launch Flow:
```bash
$ aws-vscode launch --environment web-dev

⚡ Launching VSCode Server on AWS...
✓ Creating security group
✓ Launching m7g.medium instance
✓ Waiting for instance to start
✓ Installing VSCode Server
✓ Configuring environment

🎉 VSCode Server ready!

Access: https://ec2-1-2-3-4.compute-1.amazonaws.com:8080
Password: <generated-password>

Instance ID: i-0abc123def
Region: us-east-1
Cost: ~$0.03/hour + storage
```

#### Connect Command:
```bash
$ aws-vscode connect i-0abc123def

Connecting to instance i-0abc123def...
URL: https://ec2-1-2-3-4.compute-1.amazonaws.com:8080
Password: <password-from-instance>

Opening in browser...
```

### 7. CLI Commands

All standard commands from pkg/cli:
- `launch` - Launch new VSCode instance
- `list` - Show all instances
- `status` - Instance details
- `connect` - Get connection info
- `stop` - Stop instance (preserve data)
- `terminate` - Terminate instance
- `key list/show/validate` - SSH key management

### 8. Configuration File Structure

```yaml
# environments/web-dev.yaml
name: "web-dev"
instance_type: "m7g.medium"
ami_base: "ubuntu24-arm64"
ebs_volume_size: 30

# System packages
packages:
  - git
  - curl
  - wget
  - build-essential

# Node.js setup
nodejs_version: "20"
npm_packages:
  - yarn
  - pnpm

# VSCode extensions
vscode_extensions:
  - dbaeumer.vscode-eslint
  - esbenp.prettier-vscode

# VSCode settings (optional)
vscode_settings:
  "editor.formatOnSave": true
  "editor.defaultFormatter": "esbenp.prettier-vscode"

# Environment variables
environment_vars:
  NODE_ENV: "development"
```

## Code Reuse from Monorepo

### Shared from pkg/:
- ✅ `pkg/aws` - EC2, IAM, networking, AMI selection
- ✅ `pkg/config` - State management, environment loading
- ✅ `pkg/cli` - Base CLI utilities

### VSCode-Specific (apps/vscode/internal/):
- `userdata/` - code-server setup scripts
- `cli/` - VSCode-specific CLI commands (if any)
- `config/` - VSCode-specific config handling

## Testing Strategy

### Unit Tests:
- User data template generation
- Environment config validation
- Extension list parsing

### Integration Tests:
- Launch instance with each environment
- Verify code-server is running
- Test connection to VSCode
- Verify extensions installed

### Manual Testing Checklist:
- [ ] Launch with web-dev environment
- [ ] Access VSCode in browser
- [ ] Create and edit files
- [ ] Install additional extension
- [ ] Stop and restart instance
- [ ] Verify data persistence
- [ ] Terminate and cleanup

## Timeline Estimate

- **Phase 1**: Basic structure and main.go (1 hour)
- **Phase 2**: User data script for code-server (2 hours)
- **Phase 3**: Environment configs (1 hour)
- **Phase 4**: Testing and refinement (2 hours)
- **Phase 5**: Documentation (1 hour)

**Total**: ~7 hours of focused work

## Success Criteria

✅ Users can launch VSCode Server with single command
✅ Multiple environment presets available
✅ Extensions install automatically
✅ Data persists across restarts
✅ Security groups properly configured
✅ Password displayed to user
✅ Integration with existing pkg/ infrastructure
✅ Documentation complete

## Future Enhancements

- **Custom domains**: Route53 + Let's Encrypt
- **GitHub OAuth**: Replace password auth
- **Workspace templates**: Git clone on launch
- **Extension marketplace**: Custom extension lists
- **Settings sync**: User settings persistence
- **Dev containers**: Docker-based environments
- **Remote SSH targets**: Connect to other services

## References

- code-server docs: https://coder.com/docs/code-server
- VSCode extension marketplace: https://marketplace.visualstudio.com/vscode
- Ubuntu Noble: https://releases.ubuntu.com/noble/
