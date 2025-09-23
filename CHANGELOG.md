# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
- MIT License

### Changed
- N/A

### Deprecated
- N/A

### Removed
- N/A

### Fixed
- N/A

### Security
- N/A

## [0.1.0] - 2025-01-XX

### Added
- Initial project structure and CLI framework with Cobra
- Environment configuration system with YAML support
- AWS EC2 client integration with AWS SDK v2
- Built-in environment templates (Data Science, ML PyTorch, Deep Learning, R Statistics, Computational Biology, Minimal Python)
- Environment generation from local Python setups
- Instance lifecycle management commands (launch, stop, terminate, list)
- Local state management for tracking instances
- SSH tunnel support preparation
- Auto-shutdown and hibernation configuration
- Pre-commit hooks for code quality enforcement
- GoReleaser configuration for automated releases
- Docker image support
- Homebrew tap integration
- MIT License with proper project setup