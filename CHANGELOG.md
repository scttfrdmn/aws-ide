# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### v0.6.0 Features

##### Spot Instance Support
- Add `--spot` flag to launch commands for cost-optimized instances (up to 90% savings)
- Add `--spot-max-price` flag to set maximum spot price
- Add `--spot-type` flag with options: `one-time` (default), `persistent`
- Automatic pricing defaults based on instance type
- Spot instance status displayed in `list` command output
- State tracking for spot instances

##### Enhanced List Command
- Multiple output formats: `table` (default), `json`, `simple`
- Rich table output with colors and status indicators (✓, ●, ○)
- Comprehensive instance details: ID, type, state, uptime, connection method
- Filter by state: `--state running`, `--state stopped`, `--state terminated`
- Display spot instance status and S3 sync configuration
- Human-readable uptime formatting

##### S3 Data Sync
- Add `--s3-bucket` flag for persistent workspace storage across instance lifecycle
- Add `--s3-sync-path` flag to customize sync directory (default: `/home/ubuntu/data`)
- Automatic AWS Mountpoint for S3 installation and configuration
- Architecture-aware installation (ARM64 and x86_64 support)
- Systemd mount units for automatic mounting on instance start
- Bidirectional sync for workspace persistence
- S3 bucket access testing and validation during setup
- Comprehensive logging to `/var/log/s3-sync-setup.log`
- Workspace files, settings, and SSH keys persist across stop/start cycles

##### GPU Instance Support
- Automatic GPU detection via `lspci` hardware check
- NVIDIA driver 535 installation (Ubuntu 24.04 compatible)
- CUDA Toolkit 12.2 installation for GPU computing
- Environment variable configuration (CUDA_HOME, PATH, LD_LIBRARY_PATH)
- Graceful handling when no GPU detected (skips installation silently)
- Comprehensive logging to `/var/log/gpu-setup.log`
- Works with all GPU instance types: g4dn, g5, p3, p4, etc.
- Zero configuration required - automatic setup on GPU instances

##### Config Export/Import System (Phase 1)
- Declarative YAML-based export configuration system
- Default export configs for all three IDEs:
  - `configs/rstudio-default.yaml` - RStudio settings, R packages, preferences
  - `configs/vscode-default.yaml` - VSCode settings, extensions, preferences
  - `configs/jupyter-default.yaml` - Jupyter settings, kernels, Python packages
- `export-config` command for RStudio (exports to tar.gz archive)
- Supports exporting IDE settings, package lists, dotfiles, SSH keys
- Generate commands for creating package lists on export
- Restore commands for reinstalling packages after import
- Foundation for Phase 2: S3 upload and SSM import

#### Previous Features
- **aws-rstudio Feature Parity**: Complete command set matching aws-jupyter
  - All 10 CLI commands implemented (launch, list, status, connect, stop, start, terminate, env, key, generate)
  - 4 R-specific environments: minimal, tidyverse, bioconductor, shiny
  - 27 unit tests (933 lines) covering env, generate, list, and launch commands
  - Environment management (env list, env show) fully functional
  - Key management (key list, key show, key validate) working
  - Cost optimization with idle detection and configurable timeouts
  - SSH and Session Manager connection methods
  - Public/private subnet support with NAT Gateway
  - Enhanced README reflecting v0.5.0 capabilities
- **Shared Infrastructure Testing**: Value-focused test strategy
  - Unit tests for pkg/cli utilities (formatDuration, cleanupStateFile) - 14 tests, 276 lines
  - pkg/config already at 84.7% coverage with comprehensive tests
  - Integration tests with LocalStack for AWS API validation
  - E2E tests for all 3 IDE types (Jupyter, RStudio, VSCode)
  - Proper testing pyramid: unit → integration → E2E
  - Focus on testing value over arbitrary coverage metrics
- **SSM-based readiness polling**: Secure service health checking without exposed ports
  - New `pkg/aws/ssm.go` module with SSMClient for AWS Systems Manager operations
  - `SSMClient.CheckServiceReadiness()` checks services from inside instances via curl localhost
  - `SSMClient.WaitForSSMAgent()` waits for SSM agent availability before commands
  - `PollServiceReadinessViaSSM()` in `pkg/readiness/poller.go` with progress callbacks
  - Works regardless of security group configuration (no external port access needed)
  - Uses existing IAM instance profiles for SSM access
  - Typically ready in 5-10 seconds for SSM agent, 2-3 minutes for service
- **Progress streaming enhancements**: Real-time cloud-init status during launch
  - Concurrent SSH-based progress streaming with SSM readiness polling
  - Displays cloud-init logs every 20 seconds during instance setup
  - Shows service readiness progress with elapsed time
  - Enhanced user experience with clear status updates
- **aws-vscode**: New VSCode Server (code-server) CLI tool (beta)
  - Complete CLI structure with all subcommands (launch, list, connect, stop, start, terminate, status, env, key)
  - **Full launch command implementation** with all features:
    - Environment selection (web-dev, python-dev, go-dev, fullstack)
    - Custom AMI support
    - Idle timeout configuration (default: 4h)
    - SSH and Session Manager connection methods
    - Public and private subnet support
    - NAT Gateway creation for private subnets
    - Dry-run mode to preview actions
    - Instance type override
    - Availability zone selection
  - User-data script generator for automatic code-server setup
  - 4 built-in environments: web-dev (default), python-dev, go-dev, fullstack
  - Automatic language runtime installation (Node.js 20, Python 3, Go 1.22)
  - VSCode extension auto-installation system
  - Ubuntu 22.04 Jammy LTS base OS
  - Idle detection and auto-stop system
  - SSH tunnel and Session Manager port forwarding support
  - Comprehensive README with quick start and troubleshooting
  - **Tested end-to-end**: Successfully launched instance i-0da4fbcff0a97dc0a
- Added apps/vscode to Go workspace
- Comprehensive test suite for pkg/config module (84.7% coverage)
  - environment_test.go: Environment loading, validation, listing with 7 test functions
  - state_test.go: State management, save/load cycles with 11 test functions
  - keys_test.go: SSH key storage, permissions, cleanup with 29 test functions
  - All tests use isolated temp directories with proper cleanup

### Fixed
- **aws-rstudio**: Port hardcoded incorrectly in list.go (was 8888, now correctly 8787)
  - RStudio uses port 8787, not Jupyter's port 8888
  - Fixed during unit test development

### Changed
- **All apps migrated to SSM-based readiness polling** (VSCode, Jupyter, RStudio)
  - `apps/vscode/internal/cli/launch.go`: Uses PollServiceReadinessViaSSM on port 8080
  - `apps/jupyter/internal/cli/launch.go`: Uses PollServiceReadinessViaSSM on port 8888
  - `apps/rstudio/internal/cli/launch.go`: Uses PollServiceReadinessViaSSM on port 8787
  - All apps now check service readiness from inside the instance via SSM
  - No longer depends on externally accessible service ports for health checks
  - More secure launch process with reduced security group exposure
- Updated root README to include aws-vscode
- Updated project roadmap to reflect aws-vscode alpha status

### Fixed
- golangci-lint errcheck violations in pkg/cli/delete-ami.go
- golangci-lint errcheck violation in pkg/config/environment_test.go
- All code now passes golangci-lint with zero issues
- **IAM instance profile naming conflict**: Each app now creates app-specific IAM resources
  - pkg/aws/iam.go: GetOrCreateSessionManagerRole() now accepts appPrefix parameter
  - Apps create separate roles: aws-jupyter-*, aws-vscode-*, aws-rstudio-*
  - Allows multiple IDE types to coexist without IAM resource conflicts
- **aws-vscode**: AWS CLI installation upgraded from v1 to v2
  - User-data script now installs AWS CLI v2 (required for auto-stop functionality)
  - Added architecture detection (arm64/x86_64) for correct installer selection
  - Auto-stop system requires AWS CLI v2 for `aws ec2 stop-instances` command
- **IAM propagation delays causing launch failures**: Added automatic retry logic
  - pkg/aws/ec2.go: LaunchInstance() now retries up to 5 times with exponential backoff
  - Detects IAM-related errors and waits for eventual consistency (2s, 4s, 8s, 16s delays)
  - Prevents "Invalid IAM Instance Profile" errors immediately after profile creation
  - User no longer needs to manually wait and retry launch commands
- **aws-vscode**: code-server installation failure (HOME not set)
  - Added `export HOME=/root` before running code-server installer
  - Fixes "sh: HOME: parameter not set" error in cloud-init
  - VSCode Server now installs and starts successfully on instance launch
  - Tested end-to-end: Successfully launched and connected to instance i-0ee8065b2c30a96ac

### Known Issues
- **aws-vscode**: Extensions marketplace not available
  - code-server (open-source) doesn't include Microsoft's extension marketplace
  - Extensions can be installed via command line: `code-server --install-extension <extension-id>`
  - Workaround: Use Open VSX Registry or manually install .vsix files

## [0.5.0] - 2025-01-16

### 🎉 Monorepo Transformation: Multi-IDE Platform

This release transforms aws-jupyter into AWS IDE, a monorepo supporting multiple cloud-based IDE types.

### Major Changes

#### **Monorepo Architecture**
- Transformed single-app project into Go workspace monorepo
- Created `pkg/` module for shared AWS infrastructure
- Created `apps/jupyter/` with complete aws-jupyter implementation
- Created `apps/rstudio/` with basic aws-rstudio implementation
- All apps share infrastructure while maintaining independence

#### **Code Organization**
- **Shared library (`pkg/`)**: AWS SDK integrations, CLI utilities, configuration
- **App-specific code (`apps/*/`)**: IDE-specific logic, environments, user data
- **Go workspace**: Proper module boundaries with `go.work`
- **Clean separation**: No code duplication between apps

#### **Build & CI/CD**
- Updated CI/CD pipeline for monorepo structure
- Matrix builds for pkg, jupyter, and rstudio modules
- Separate test, lint, and build jobs for each component
- All builds and tests passing

#### **Documentation**
- Updated root README for multi-IDE overview
- Created comprehensive RStudio README
- Updated ROADMAP for multi-IDE roadmap (v0.5.0-v1.0.0)
- Consolidated docs into app-specific directories
- Removed duplicate documentation

#### **aws-rstudio (New)**
- Basic implementation with core commands
- Shares all infrastructure with Jupyter
- Supports launch, list, status, connect, stop, terminate
- SSH and Session Manager connection methods
- Public/private subnet support
- Feature parity work in progress (see ROADMAP)

### Added
- **aws-rstudio CLI**: New command-line tool for RStudio Server
- **Shared pkg/ library**: Reusable AWS infrastructure code
- **Go workspace**: Multi-module project structure
- **Monorepo CI/CD**: Matrix builds for all modules
- **RStudio README**: Complete documentation for RStudio launcher

### Changed
- **Project name**: aws-jupyter → AWS IDE (aws-ide)
- **Repository structure**: Single app → Monorepo
- **Code location**: `internal/` → `pkg/` (shared) and `apps/*/internal/` (app-specific)
- **Documentation**: Root docs → `apps/*/docs/`
- **Build process**: Single binary → Multiple app binaries
- **Version strategy**: Shared version across all apps

### Fixed
- Test failures in apps/jupyter/internal/cli/launch_test.go (function signature mismatches)
- Build artifacts not ignored by git
- CI/CD pipeline incompatible with workspace structure

### Removed
- Legacy `internal/` directory (moved to `pkg/`)
- Legacy `cmd/` directory (moved to `apps/*/cmd/`)
- Root-level `go.mod` and `go.sum` (using `go.work` instead)
- Duplicate documentation in root `docs/` directory

### Migration Notes

**For Existing Users:**
- `aws-jupyter` functionality unchanged - all features preserved
- Binary location changed: `./aws-jupyter` → `./apps/jupyter/aws-jupyter`
- Install path unchanged: `/usr/local/bin/aws-jupyter`
- Configuration compatible: `~/.aws-jupyter/` still used
- State files compatible: No migration needed

**For Developers:**
- Update imports: `github.com/scttfrdmn/aws-jupyter/internal/...` → `github.com/scttfrdmn/aws-ide/pkg/...`
- Build from app directory: `cd apps/jupyter && go build ./cmd/aws-jupyter`
- Run tests per module: `cd pkg && go test ./...` or `cd apps/jupyter && go test ./...`
- CI/CD uses matrix builds for each module

### Metrics
- **Modules**: 3 (pkg, jupyter, rstudio)
- **Test Coverage**: 18.7% overall (unchanged)
- **Build Status**: All modules building successfully
- **Binary Size**: ~44MB per app
- **IDE Support**: 2 types (Jupyter Lab, RStudio Server)

### Looking Forward

This monorepo transformation enables:
- Easy addition of new IDE types (VSCode, JupyterHub, etc.)
- Shared infrastructure reduces duplication
- Consistent behavior across all IDE types
- Independent versioning possible in future

See [ROADMAP.md](ROADMAP.md) for v0.5.0-v1.0.0 planning.

---

## [0.2.0] - 2025-01-14

### 🎉 Major Release: Production-Ready with Complete Lifecycle Management

This release marks a significant milestone with comprehensive code quality improvements, full CLI implementation, enhanced test coverage, and complete documentation.

### Added

#### **Phase 1: Code Quality & Refactoring**
- Complexity reduction across codebase (reduced cyclomatic complexity)
- Comprehensive inline documentation and code comments
- Advanced linting with golangci-lint (strictness improvements)
- Pre-commit hooks configuration for code quality enforcement
- GitHub Actions CI workflow with automated testing
- Multi-version Go testing (1.22 and 1.23)
- Code coverage reporting with Codecov integration
- GoReleaser configuration for cross-platform releases (Linux, macOS, Windows)

#### **Phase 2: Code Improvements**
- Constants extraction for all magic numbers and strings
- Removed unused code and dead imports
- Standardized error messages across packages
- Improved code organization and readability

#### **Phase 3: Feature Completion - All CLI Commands**
- `launch` - Launch new Jupyter Lab instances with full configuration
- `list` - Display all running instances with status information
- `status` - Detailed instance information and health checks
- `connect` - Connect to existing instances via SSH or Session Manager
- `stop` - Stop instances (preserves EBS volumes)
- `terminate` - Terminate instances (cleanup and resource deletion)
- `key list` - View local and AWS key pairs
- `key show` - Display default key details
- `key validate` - Check key file permissions
- `key cleanup` - Remove orphaned keys with dry-run support
- `generate` - Create custom environments from local Python setups

#### **Phase 4: Test Coverage Improvements**
- AWS package test suite (networking, IAM, security, key pairs, AMI selection)
- 322% improvement in AWS package coverage (0.9% → 3.8%)
- 33% improvement in overall coverage (14% → 18.7%)
- CLI package coverage: 27.8%
- Config package coverage: 19.1%
- Comprehensive struct validation tests
- Business logic and naming convention tests

#### **Infrastructure & Networking**
- Session Manager support for SSH-less instance access
- IAM role and instance profile management for Session Manager
- Private subnet support with optional NAT Gateway
- Advanced networking configuration options
- VPC subnet selection (public/private)
- NAT Gateway creation and route table management
- Connection method selection (SSH or Session Manager)
- Security group customization based on connection method
- SSH key pair management with economical reuse strategy
- Regional key pair naming (aws-jupyter-{region})
- Secure local key storage with proper permissions (600)

#### **Environment System**
- User data script generator for automated Jupyter Lab setup
- Dynamic AMI selection (Ubuntu and Amazon Linux)
- Support for arm64 and x86_64 architectures
- Automated package installation (system and Python packages)
- Jupyter Lab extensions installation and configuration
- Systemd service creation for Jupyter Lab

#### **Documentation**
- Complete [ROADMAP.md](ROADMAP.md) with v0.2.0 through v1.0.0 planning
- [Session Manager Setup Guide](docs/SESSION_MANAGER_SETUP.md) - Complete SSM configuration
- [Private Subnet Guide](docs/PRIVATE_SUBNET_GUIDE.md) - Best practices and cost analysis
- [Troubleshooting Guide](docs/TROUBLESHOOTING.md) - Common issues and solutions
- [Examples & Use Cases](docs/EXAMPLES.md) - 20 real-world scenarios
- Updated README.md with all new features and commands
- Removed "UNDER ACTIVE DEVELOPMENT" warning - production ready!

### Changed
- Migrated from MIT to Apache 2.0 license
- Enhanced launch command with comprehensive networking flags
- Updated security groups to support both SSH and Session Manager
- Improved dry-run output with detailed action plans and cost estimates
- Improved error handling for AWS API calls with detailed messages
- Standardized struct field names (Arn, DefaultPrefix, Region)
- Optimized test execution speed (no AWS API dependencies in unit tests)

### Fixed
- Test function signatures in launch_test.go
- Formatting violations across all source files
- NAT Gateway API field name (Filters -> Filter)
- Struct field naming inconsistencies (ARN → Arn)
- KeyPairStrategy field names (DefaultName → DefaultPrefix + Region)
- Ineffectual assignment in networking_test.go
- All linting issues across codebase

### Performance
- Fast unit tests (<1s execution, no network calls)
- Optimized AWS API call patterns
- Efficient resource reuse (NAT Gateway, security groups, key pairs)

### Security
- Session Manager support eliminates SSH key exposure
- Private subnet support for enhanced network isolation
- IAM role-based access control
- Audit logging through CloudTrail integration
- Secure key storage with proper file permissions

### Metrics
- **Test Coverage**: 18.7% overall (AWS: 3.8%, CLI: 27.8%, Config: 19.1%)
- **Code Quality**: A+ Go Report Card
- **Commands**: 10 complete CLI commands
- **Documentation**: 5 comprehensive guides
- **Built-in Environments**: 6 pre-configured templates

## [0.1.0] - 2025-01-13

### Added
- Initial CLI structure with Cobra framework
- Environment configuration system with YAML support
- AWS EC2 client integration with AWS SDK v2
- Built-in environment templates:
  - Data Science (pandas, numpy, matplotlib, scikit-learn)
  - ML PyTorch (PyTorch, transformers, datasets)
  - Deep Learning (PyTorch, TensorFlow, MLflow, Optuna)
  - R Statistics (R kernel, tidyverse)
  - Computational Biology (biopython, samtools, bedtools)
  - Minimal Python (basic setup)
- Environment generation from local Python setups
- Instance lifecycle management (launch, stop, terminate, list)
- Local state management for tracking instances
- SSH tunnel support preparation
- Auto-shutdown and hibernation configuration
- Pre-commit hooks for code quality
- GitHub issue templates (bug report, feature request, question)
- Pull request template
- Comprehensive README with installation and usage instructions
- Project documentation and contributing guidelines

[Unreleased]: https://github.com/scttfrdmn/aws-ide/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/scttfrdmn/aws-ide/compare/v0.2.0...v0.5.0
[0.2.0]: https://github.com/scttfrdmn/aws-ide/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/scttfrdmn/aws-ide/releases/tag/v0.1.0