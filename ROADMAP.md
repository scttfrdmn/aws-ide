# AWS IDE Roadmap

This document outlines the future development plans for the AWS IDE project. The project follows semantic versioning and is organized into phases.

## 📊 Current Status: Monorepo Established

**Project Evolution:**
- ✅ **v0.1.0-v0.4.0**: aws-jupyter (single app) - Feature complete
- ✅ **Monorepo Migration**: Transformed into multi-IDE platform (January 2025)
- 🚀 **Current Focus**: RStudio feature parity and shared infrastructure improvements

**Monorepo Structure:**
- `pkg/` - Shared AWS infrastructure library
- `apps/jupyter/` - Jupyter Lab launcher (feature complete)
- `apps/rstudio/` - RStudio Server launcher (basic implementation)

**Coverage Status:**
- pkg: Tests passing (aws package tested)
- apps/jupyter: 27.8% CLI, all tests passing
- apps/rstudio: No tests yet (needs development)

---

## 🎯 v0.5.0 - Monorepo Stabilization (Current - Q1 2025)

**Status:** In Progress
**Target Date:** February 2025
**Priority:** CRITICAL

### Goals
Complete the monorepo migration and establish RStudio as a first-class citizen alongside Jupyter.

### Tasks

**Completed:**
- [x] Transform codebase into Go workspace
- [x] Extract shared infrastructure into pkg/
- [x] Create apps/jupyter with full feature set
- [x] Create apps/rstudio with basic implementation
- [x] Fix all test failures
- [x] Update CI/CD for monorepo structure
- [x] Remove legacy code and duplication
- [x] Update documentation for monorepo
- [x] **SSM-based readiness polling** for secure service health checks
  - [x] Created pkg/aws/ssm.go with SSMClient implementation
  - [x] Added PollServiceReadinessViaSSM() to pkg/readiness/poller.go
  - [x] Migrated VSCode to SSM polling (port 8080)
  - [x] Migrated Jupyter to SSM polling (port 8888)
  - [x] Migrated RStudio to SSM polling (port 8787)
  - [x] All apps tested end-to-end with SSM readiness checks
  - [x] Works regardless of security group configuration
- [x] **Progress streaming improvements**
  - [x] Real-time cloud-init progress via SSH
  - [x] Concurrent progress streaming and SSM polling
  - [x] Enhanced user experience during launch

**Completed:**
- [x] **RStudio Feature Parity**
  - [x] Ported all 10 commands from Jupyter to RStudio
  - [x] Created 4 R-specific environments (minimal, tidyverse, bioconductor, shiny)
  - [x] Added 933 lines of unit tests (27 test functions)
  - [x] Fixed port configuration bug (8888 → 8787)
  - [x] Updated README with current capabilities
  - [x] Verified all commands work end-to-end

**Completed:**
- [x] **Shared Infrastructure Testing** (Value-Focused Approach)
  - [x] Unit tests for pure functions (formatDuration, cleanupStateFile, etc.)
  - [x] Config management fully tested (environment, state, keys)
  - [x] Integration tests for AWS operations (EC2, IAM, AMI, SSM)
  - [x] E2E tests for complete workflows (all 3 IDE types)
  - [x] Testing pyramid properly balanced: unit → integration → E2E

**In Progress:**
- [ ] Documentation
  - [ ] Update CHANGELOG for monorepo
  - [ ] Create migration guide for users

### Success Criteria
- ✅ Both apps build and test successfully
- ✅ CI/CD pipeline working for monorepo
- ✅ RStudio has feature parity with Jupyter
- ✅ Shared library appropriately tested (value over coverage metrics)
- ⏳ Complete documentation for both apps

---

## 🧪 v0.6.0 - Testing & Quality (Q1 2025)

**Status:** In Progress
**Target Date:** March 2025
**Priority:** HIGH

### Goals
Establish comprehensive testing infrastructure with unit, integration, and E2E tests.

### Tasks

**Completed:**
- [x] **Test Infrastructure**
  - [x] Created comprehensive Makefile with test targets (unit, integration, smoke, e2e, regression)
  - [x] Set up LocalStack Docker Compose configuration for AWS mocking
  - [x] Added build tags for test separation (-tags=integration, -tags=smoke, -tags=e2e)
  - [x] Configured appropriate timeouts for each test type

- [x] **Unit Tests for SSM and Readiness**
  - [x] pkg/aws/ssm_test.go (354 lines, 14 test functions)
  - [x] pkg/readiness/poller_test.go (461 lines, 12 test functions)
  - [x] Tests for CommandResult struct, ServiceConfig, polling configuration

- [x] **Integration Tests with LocalStack**
  - [x] pkg/aws/ec2_integration_test.go (602 lines, 12 test functions)
  - [x] Tests for EC2Client and SSMClient initialization
  - [x] Verified LocalStack compatibility
  - [x] Integration tests run with `make test-integration`
  - [x] EC2 LaunchInstance tests (valid params, auto-subnet, invalid params)
  - [x] Instance operations tests (start/stop/terminate/get info)
  - [x] AMI operations tests (create/list/delete)
  - [x] IAM role tests (GetOrCreateSessionManagerRole with idempotency)
  - [x] Error handling tests (non-existent resources, invalid params)
  - [x] AMI selector integration tests (multiple OS/arch combinations)

- [x] **Smoke Tests (Quick Real AWS Validation)**
  - [x] pkg/aws/ec2_smoke_test.go (370 lines, 7 test functions)
  - [x] Basic EC2 client creation and connectivity
  - [x] IAM role verification (Session Manager)
  - [x] Subnet discovery (public subnets)
  - [x] Instance type availability checks
  - [x] AMI discovery (Ubuntu 24.04, Amazon Linux)
  - [x] Availability zone compatibility
  - [x] Quick launch prerequisite validation
  - [x] All tests passing with AWS_PROFILE=aws
  - [x] Run with `make test-smoke`

- [x] **End-to-End Tests**
  - [x] pkg/e2etest/helpers.go (500+ lines) - Shared E2E test infrastructure
  - [x] Complete launch → connect → terminate flow (Jupyter) - 4 test functions
  - [x] Complete launch → connect → terminate flow (RStudio) - 4 test functions
  - [x] Complete launch → connect → terminate flow (VSCode) - 5 test functions
  - [x] Test Session Manager connection methods (UseSessionMgr flag)
  - [x] Test environment generation and customization (MultipleEnvironments tests)
  - [x] Instance lifecycle testing (stop/start operations)
  - [x] Multi-architecture testing (ARM64 Graviton + x86)
  - [x] Custom port configuration testing
  - [x] SSM-based readiness polling (no security group dependency)
  - [x] Automatic cleanup with defer statements
  - [x] Run with `make test-e2e`

- [x] **Documentation**
  - [x] Updated TESTING.md with LocalStack setup instructions
  - [x] Documented testing strategy pyramid (unit → integration → smoke → E2E)
  - [x] Added LocalStack vs Moto comparison
  - [x] Documented SSM readiness polling end-to-end test results

**Current Coverage Status:**
- pkg/aws: 2.5% (struct tests only; most functions require AWS/LocalStack)
- pkg/cli: 0.0% (Cobra commands; hard to unit test without extensive mocking)
- pkg/config: 84.7% (✅ already well-tested)
- pkg/readiness: 0.0% (struct tests only; polling functions require network)
- apps/jupyter/cli: 27.8%
- apps/rstudio/cli: 0.0%

**Testing Strategy:**
- **Unit Tests**: Structs, data validation, pure functions (no AWS/network)
- **Integration Tests**: AWS API calls via LocalStack (no real AWS costs)
- **Smoke Tests**: Quick real AWS validation (<10 min, minimal cost)
- **E2E Tests**: Full workflows with real AWS (longer, higher cost)

### Success Criteria
- ✅ Comprehensive Makefile with all test types
- ✅ LocalStack integration working
- ✅ Unit tests for testable modules
- ✅ Smoke tests for quick validation
- ✅ Integration tests covering major AWS operations
- ✅ E2E tests for all three apps (Jupyter, RStudio, VSCode)
- ⏳ CI/CD running unit + integration tests automatically

---

## ✨ v0.7.0 - User Experience (Q2 2025)

**Status:** Planned
**Target Date:** April 2025
**Priority:** MEDIUM

### Goals
Improve usability and developer experience with better UI/UX.

### Tasks
- [ ] Interactive Launch Wizard
  - [ ] Guide users through launch options
  - [ ] Provide recommendations based on use case
  - [ ] Show cost estimates
  - [ ] Support both Jupyter and RStudio

- [ ] Enhanced Output
  - [ ] Add color-coded output (success/warning/error)
  - [ ] Add progress bars for long operations
  - [ ] Improve table formatting in `list` command
  - [ ] Add spinner for API calls

- [ ] Better Error Messages
  - [ ] Contextual error messages
  - [ ] Suggest fixes for common errors
  - [ ] Add error codes for documentation
  - [ ] Include AWS error details when relevant

- [ ] Configuration File Support
  - [ ] `~/.aws-ide/config.yaml` for defaults
  - [ ] Profile support (dev, prod, etc.)
  - [ ] Per-app configuration sections
  - [ ] Validate configuration on load

### Success Criteria
- ✅ Interactive wizard working
- ✅ Color output on supported terminals
- ✅ Error messages provide actionable guidance
- ✅ Unified configuration system

---

## 💰 v0.8.0 - Cost Management (Q2 2025)

**Status:** Planned
**Target Date:** May 2025
**Priority:** MEDIUM

### Goals
Add cost tracking, monitoring, and optimization features.

### Tasks
- [ ] Cost Tracking
  - [ ] Track instance running time
  - [ ] Calculate estimated costs per instance
  - [ ] Show cumulative costs in `list` command
  - [ ] Export cost reports (CSV/JSON)
  - [ ] Support both Jupyter and RStudio instances

- [ ] Cost Optimization
  - [ ] Recommend cheaper instance types
  - [ ] Enhanced idle detection (already started)
  - [ ] Auto-stop improvements
  - [ ] Suggest Spot instances
  - [ ] Cost estimation in dry-run mode

- [ ] Monitoring Dashboard
  - [ ] Instance health checks
  - [ ] Resource utilization tracking (CPU, memory)
  - [ ] Alert on unexpected costs
  - [ ] Unified dashboard for all IDE types

### Success Criteria
- ✅ Accurate cost tracking per instance
- ✅ Cost optimization recommendations
- ✅ Monitoring dashboard functional
- ✅ Works for all IDE types

---

## 🔄 v0.9.0 - Multi-Instance Management (Q3 2025)

**Status:** Planned
**Target Date:** July 2025
**Priority:** MEDIUM

### Goals
Improve management of multiple instances and add batch operations.

### Tasks
- [ ] Batch Operations
  - [ ] Stop/start/terminate multiple instances
  - [ ] Filter by IDE type, environment, tags
  - [ ] Pattern matching for instance selection
  - [ ] Bulk operations with confirmation

- [ ] Instance Groups
  - [ ] Group instances by project/team
  - [ ] Manage groups together
  - [ ] Share groups across team
  - [ ] Group-level policies

- [ ] Enhanced List Command
  - [ ] Filter by IDE type (jupyter, rstudio, etc.)
  - [ ] Filter by state, environment, age
  - [ ] Sort options (cost, uptime, name)
  - [ ] Export to JSON/CSV

### Success Criteria
- ✅ Batch operations working reliably
- ✅ Filtering and sorting feature-rich
- ✅ Multi-instance management efficient
- ✅ Works across all IDE types

---

## 🆕 v0.10.0 - Additional IDEs (Q3 2025)

**Status:** Planned
**Target Date:** September 2025
**Priority:** LOW-MEDIUM

### Goals
Add support for additional cloud-based IDE types.

### Tasks
- [ ] VSCode Server
  - [ ] Create apps/vscode with basic implementation
  - [ ] Port core features from Jupyter/RStudio
  - [ ] VSCode-specific environments
  - [ ] Extension management support

- [ ] JupyterHub (Multi-user)
  - [ ] Create apps/jupyterhub
  - [ ] Multi-user authentication
  - [ ] User management
  - [ ] Resource quotas

- [ ] Additional Candidates
  - [ ] Theia IDE
  - [ ] Code-server alternatives
  - [ ] Zeppelin notebooks
  - [ ] Custom IDE support framework

### Success Criteria
- ✅ At least 2 additional IDE types supported
- ✅ All core features work across all IDE types
- ✅ Documentation for each IDE type
- ✅ Unified CLI experience

---

## 🏢 v0.11.0 - Enterprise Features (Q4 2025)

**Status:** Planned
**Target Date:** November 2025
**Priority:** LOW

### Goals
Add features for enterprise/team usage.

### Tasks
- [ ] Multi-Account Support
  - [ ] Switch between AWS accounts
  - [ ] Cross-account resource access
  - [ ] Organization-wide policies

- [ ] Team Collaboration
  - [ ] Share instances with team members
  - [ ] Role-based access control
  - [ ] Audit logging
  - [ ] Team resource quotas

- [ ] Compliance & Security
  - [ ] Encryption at rest (EBS)
  - [ ] Compliance reporting
  - [ ] Security scanning
  - [ ] SOC2/HIPAA considerations

### Success Criteria
- ✅ Multi-account management working
- ✅ Team features functional
- ✅ Compliance requirements met

---

## 🚀 v1.0.0 - Production Ready (Q4 2025)

**Status:** Planned
**Target Date:** December 2025
**Priority:** HIGH

### Goals
Achieve production-grade stability and completeness.

### Requirements for v1.0.0
- [ ] **Test Coverage:** 60%+ overall coverage
- [ ] **Documentation:** Complete and up-to-date for all IDEs
- [ ] **Performance:** <2s command response time
- [ ] **Stability:** No critical bugs in issue tracker
- [ ] **Security:** Security audit completed
- [ ] **Compatibility:** Tested on Linux/macOS/Windows
- [ ] **IDE Support:** 3+ IDE types fully supported
- [ ] **User Base:** 100+ active users
- [ ] **Feedback:** Incorporate feedback from beta users

### Final Polish
- [ ] Performance optimization
- [ ] Memory usage optimization
- [ ] Binary size optimization
- [ ] Comprehensive benchmarking
- [ ] Professional branding/logo
- [ ] Video tutorials for each IDE type
- [ ] Migration guides

### Success Criteria
- ✅ All v1.0.0 requirements met
- ✅ Production deployments successful
- ✅ Positive user feedback
- ✅ Active community

---

## 📋 Backlog & Ideas

Features under consideration but not yet scheduled:

### Additional IDE Support
- [ ] PyCharm Server
- [ ] IntelliJ IDEA Server
- [ ] RStudio Connect
- [ ] Observable Framework
- [ ] Quarto publishing platform

### Advanced Features
- [ ] GPU instance support
- [ ] Auto-scaling based on load
- [ ] Kubernetes deployment option
- [ ] Container-based environments
- [ ] Spot instance support
- [ ] Reserved instance recommendations

### Integrations
- [ ] Git integration for version control
- [ ] Database connection management
- [ ] S3 data sync
- [ ] Secrets management (AWS Secrets Manager)
- [ ] Parameter Store integration
- [ ] GitHub Codespaces-like experience

### Developer Tools
- [ ] Plugin system for extensibility
- [ ] REST API server
- [ ] Web UI dashboard (unified across IDEs)
- [ ] Mobile app (iOS/Android)
- [ ] VS Code extension
- [ ] Terraform provider

---

## 🤝 Contributing

We welcome contributions! Here's how to get involved:

1. **Check the roadmap** - Find features you're interested in
2. **Open an issue** - Discuss the feature before implementing
3. **Submit a PR** - Follow our [contributing guidelines](CONTRIBUTING.md)
4. **Join discussions** - Participate in feature planning

### Priority Areas for Contributions
- RStudio feature parity (v0.5.0)
- Testing infrastructure (v0.6.0)
- Additional IDE support (v0.10.0)
- Documentation and examples
- Bug fixes and performance improvements

---

## 📅 Release Cycle

- **Major versions** (x.0.0): Significant new features, possible breaking changes
- **Minor versions** (0.x.0): New features, backward compatible
- **Patch versions** (0.0.x): Bug fixes, documentation updates

**Typical Timeline:**
- Minor releases: Every 6-8 weeks
- Patch releases: As needed
- Major releases: When significant milestones achieved

---

## 📊 Metrics & Goals

### Project Health Metrics
- Test Coverage: Target 60% by v1.0.0 (currently 18.7%)
- Response Time: <2s for all commands
- Binary Size: <50MB for all platforms (currently ~44MB)
- Memory Usage: <100MB for typical operations
- Startup Time: <100ms

### Community Metrics
- Active Users: 1000+ by v1.0.0
- GitHub Stars: 500+ by v1.0.0
- Contributors: 10+ by v1.0.0
- Issues Resolved: 90%+ within 30 days
- IDE Types Supported: 3+ by v1.0.0

### Monorepo Health
- Shared library stability: No breaking changes
- Cross-IDE compatibility: All features work identically
- Documentation coverage: 100% of features documented
- CI/CD reliability: 99%+ green builds

---

## 🔄 Versioning Strategy

We follow [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR** version when making incompatible API changes
- **MINOR** version when adding functionality in a backward compatible manner
- **PATCH** version when making backward compatible bug fixes

**Special Considerations for Monorepo:**
- Version numbers apply to the entire project (all apps share version)
- Breaking changes in pkg/ trigger major version bump
- New IDE support triggers minor version bump
- Bug fixes in any app trigger patch version bump

---

## 📝 Last Updated

- **Date:** January 2025
- **Version:** v0.5.0 (Monorepo Migration - In Progress)
- **Next Review:** February 2025
- **Project Status:** Transitioning from single app to multi-IDE platform

---

## 💬 Feedback

Have suggestions for the roadmap?
- Open an issue: https://github.com/scttfrdmn/aws-ide/issues
- Start a discussion: https://github.com/scttfrdmn/aws-ide/discussions

We prioritize features based on:
1. User demand
2. Multi-IDE compatibility
3. Project goals
4. Implementation complexity
5. Maintainability
6. Community contributions
