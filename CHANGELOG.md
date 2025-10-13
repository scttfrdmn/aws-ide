# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Apache 2.0 license with copyright 2025 Scott Friedman
- User data script generator for automated Jupyter Lab environment setup
- Dynamic AMI selection with support for Ubuntu and Amazon Linux
- Support for arm64 and x86_64 architectures
- Automated package installation (system and Python packages)
- Jupyter Lab extensions installation and configuration
- Systemd service creation for Jupyter Lab
- GitHub Actions CI workflow with automated testing
- Multi-version Go testing (1.22 and 1.23)
- Code coverage reporting with Codecov integration
- Linting pipeline with gofmt, go vet, and golangci-lint
- GoReleaser configuration for cross-platform releases
- Automated releases for Linux, macOS, and Windows (amd64/arm64)
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
- Key pair CLI commands (list, show, validate, cleanup)

### Changed
- Migrated from MIT to Apache 2.0 license
- Enhanced launch command with networking flags
- Updated security groups to support Session Manager
- Improved dry-run output with detailed action plans
- Improved error handling for AWS API calls

### Fixed
- Test function signatures in launch_test.go
- Formatting violations across all source files
- NAT Gateway API field name (Filters -> Filter)

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

[Unreleased]: https://github.com/scttfrdmn/aws-jupyter/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/scttfrdmn/aws-jupyter/releases/tag/v0.1.0