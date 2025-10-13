# AWS-Jupyter Project Remediation Plan

**Date:** 2025-01-13
**Objective:** Achieve A+ Go Report Card rating, production-ready code quality, and comprehensive test coverage
**Target Completion:** 5-7 days (phased approach)

---

## Executive Summary

**Current State:**
- ✅ All tests passing (45 tests)
- ⚠️ Test coverage: 19.9% (target: 70%+)
- ⚠️ Linting issues: 17 errors (golangci-lint)
- ⚠️ Cyclomatic complexity: 1 function over threshold (49 > 15)
- ⚠️ Go style issues: ~30 golint warnings
- ⚠️ Ineffectual assignments: 1 issue
- ❌ No E2E tests
- ❌ Critical AWS infrastructure code untested (0.9% coverage)

**Go Report Card Requirements (A+ Grade):**
- gofmt: 100% ✅ (currently passing)
- go vet: 100% ✅ (currently passing)
- gocyclo: <15 complexity ⚠️ (1 violation: runLaunch = 49)
- golint: 100% ⚠️ (~30 issues)
- ineffassign: 100% ⚠️ (1 issue)
- misspell: 100% ✅ (currently passing)
- license: 100% ✅ (Apache 2.0 present)

**Target State:**
- 🎯 Go Report Card: A+ (99%+)
- 🎯 Test Coverage: 70%+ overall, 80%+ critical paths
- 🎯 Zero linting issues
- 🎯 All functions < 15 cyclomatic complexity
- 🎯 Full documentation on all exported symbols
- 🎯 E2E test suite for critical workflows
- 🎯 Production-ready error handling

---

## Phase 1: Critical Fixes (Day 1) - BLOCKING

**Priority: P0 - Must complete before any other work**

### 1.1 Fix All Linting Errors (17 issues)

**errcheck violations (16 issues) - Production Code:**

```go
// internal/aws/iam.go:219
if err := i.client.DeleteInstanceProfile(ctx, &iam.DeleteInstanceProfileInput{
    InstanceProfileName: aws.String(name),
}); err != nil {
    return fmt.Errorf("failed to delete instance profile: %w", err)
}

// internal/aws/networking.go:149
if err := e.client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
    AllocationId: aws.String(*allocation.AllocationId),
}); err != nil {
    return fmt.Errorf("failed to release elastic IP: %w", err)
}

// internal/aws/security.go:174
if err := e.client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
    GroupId: aws.String(sgId),
}); err != nil {
    return fmt.Errorf("failed to delete security group: %w", err)
}

// internal/config/keys.go:137
// Add comment explaining intentional ignore, or handle error:
if err := os.Remove(pubKeyPath); err != nil && !os.IsNotExist(err) {
    // Log warning but don't fail - public key cleanup is best-effort
}
```

**errcheck violations - Test Files:**
```go
// internal/aws/ec2_test.go - Add proper cleanup:
if err := os.Setenv("AWS_REGION", "us-west-2"); err != nil {
    t.Fatalf("failed to set env: %v", err)
}

// internal/cli/*_test.go - Check pipe operations:
if err := w.Close(); err != nil {
    t.Fatalf("failed to close pipe: %v", err)
}
if _, err := buf.ReadFrom(r); err != nil {
    t.Fatalf("failed to read output: %v", err)
}
```

**ineffassign violation:**
```go
// internal/aws/ami.go:102 - Remove unused assignment:
// Delete line: namePattern := "amzn2-ami-hvm-*"
// This variable is assigned but never used
```

**Estimated Time:** 2-3 hours
**Complexity:** Low
**Risk:** Low

---

### 1.2 Fix Cyclomatic Complexity (runLaunch = 49)

**Problem:** `internal/cli/launch.go:46` - runLaunch function has complexity 49 (threshold: 15)

**Solution:** Refactor into smaller, focused functions:

```go
// Before: 366-line monolithic function
func runLaunch(environment, instanceType, idleTimeout, profile, region string,
               dryRun bool, connectionMethod, subnetType string, createNatGateway bool) error {
    // 366 lines of code...
}

// After: Break into logical phases
func runLaunch(...) error {
    env, err := loadAndValidateEnvironment(environment, instanceType)
    if err != nil {
        return err
    }

    if dryRun {
        return executeDryRun(env, profile, region, connectionMethod, subnetType, createNatGateway)
    }

    return executeLaunch(env, profile, region, idleTimeout, connectionMethod, subnetType, createNatGateway)
}

func loadAndValidateEnvironment(environment, instanceType string) (*config.Environment, error) {
    // Lines 48-66
}

func executeDryRun(env *config.Environment, profile, region, connectionMethod, subnetType string, createNatGateway bool) error {
    // Lines 84-169 - dry run logic
}

func executeLaunch(env *config.Environment, profile, region, idleTimeout, connectionMethod, subnetType string, createNatGateway bool) error {
    // Split further:
    ctx := context.Background()

    // Setup phase
    ec2Client, actualRegion, err := setupAWSClient(ctx, profile, region)
    if err != nil {
        return err
    }

    // Authentication phase
    keyInfo, instanceProfile, err := setupAuthentication(ctx, ec2Client, profile, actualRegion, connectionMethod)
    if err != nil {
        return err
    }

    // Network phase
    subnet, natGateway, err := setupNetworking(ctx, ec2Client, subnetType, createNatGateway)
    if err != nil {
        return err
    }

    // Security phase
    securityGroup, err := setupSecurity(ctx, ec2Client, subnet.VpcId, connectionMethod)
    if err != nil {
        return err
    }

    // Launch phase
    return launchAndDisplayInstance(ctx, ec2Client, env, actualRegion, subnet, securityGroup, keyInfo, instanceProfile, connectionMethod)
}

// Each helper function should be < 50 lines and < 15 complexity
```

**Target:** Each function < 15 cyclomatic complexity, < 100 lines

**Estimated Time:** 3-4 hours
**Complexity:** Medium
**Risk:** Medium (requires careful refactoring, extensive testing)

---

### 1.3 Fix Go Lint Issues (~30 violations)

**Problem Categories:**

**1. Missing documentation on exported symbols:**
```go
// EC2Client manages AWS EC2 operations for launching and managing instances.
// It provides methods for instance lifecycle management, networking, and security.
type EC2Client struct {
    client *ec2.Client
    region string
}

// NewEC2Client creates a new EC2 client using the specified AWS profile.
// It loads AWS configuration and initializes the EC2 service client.
// Returns an error if AWS credentials cannot be loaded or are invalid.
func NewEC2Client(ctx context.Context, profile string) (*EC2Client, error) {
    // ...
}

// LaunchInstance creates and starts a new EC2 instance with the specified configuration.
// It handles subnet selection, user data encoding, and tag application.
// Returns the launched instance or an error if the operation fails.
func (e *EC2Client) LaunchInstance(ctx context.Context, params LaunchParams) (*types.Instance, error) {
    // ...
}
```

**2. ID suffix violations (Id → ID):**
```go
// Before:
type SubnetInfo struct {
    SubnetId     string  // ❌
    VpcId        string  // ❌
}

var subnetId string // ❌

// After:
type SubnetInfo struct {
    SubnetID     string  // ✅
    VpcID        string  // ✅
}

var subnetID string // ✅
```

**Files requiring fixes:**
- `internal/aws/ec2.go` - 8 issues
- `internal/aws/networking.go` - 15 issues
- `internal/aws/security.go` - 7 issues

**Estimated Time:** 2-3 hours
**Complexity:** Low (mostly documentation)
**Risk:** Low

---

## Phase 2: AWS Infrastructure Testing (Days 2-3) - CRITICAL

**Priority: P1 - Required for production readiness**

### 2.1 Add Unit Tests for AWS Infrastructure (0.9% → 60%+)

**Target Files (currently 0% coverage):**

#### 2.1.1 AMI Selection (`internal/aws/ami.go`)
```go
// internal/aws/ami_test.go
func TestAMISelector_GetDefaultAMI(t *testing.T) {
    tests := []struct {
        name   string
        region string
        want   string
    }{
        {"us-east-1", "us-east-1", "ami-0c55b159cbfafe1f0"},
        {"eu-west-1", "eu-west-1", "ami-0d71ea30463e0ff8d"},
    }
    // ...
}

func TestAMISelector_FindUbuntuAMI_Mock(t *testing.T) {
    // Use AWS SDK mocking
    // Test happy path, no results, API errors
}
```

#### 2.1.2 IAM Role Management (`internal/aws/iam.go`)
```go
// internal/aws/iam_test.go
func TestIAMClient_GetOrCreateSessionManagerRole(t *testing.T) {
    // Test: role exists, role creation, profile attachment
}

func TestIAMClient_RoleExists(t *testing.T) {
    // Test: exists, doesn't exist, API error
}
```

#### 2.1.3 Key Pair Management (`internal/aws/keypair.go`)
```go
// internal/aws/keypair_test.go
func TestKeyPairStrategy_GetDefaultKeyName(t *testing.T) {
    tests := []struct {
        name     string
        strategy KeyPairStrategy
        want     string
    }{
        {
            name: "default strategy",
            strategy: DefaultKeyPairStrategy("us-west-2"),
            want: "aws-jupyter-us-west-2",
        },
    }
    // ...
}

func TestEC2Client_GetOrCreateKeyPair(t *testing.T) {
    // Test: exists and reuse, create new, creation failure
}
```

#### 2.1.4 Networking (`internal/aws/networking.go`)
```go
// internal/aws/networking_test.go
func TestEC2Client_GetSubnet(t *testing.T) {
    // Test: public subnet, private subnet, VPC override
}

func TestEC2Client_GetOrCreateNATGateway(t *testing.T) {
    // Test: exists, create new, wait for available
}
```

#### 2.1.5 Security Groups (`internal/aws/security.go`)
```go
// internal/aws/security_test.go
func TestSecurityGroupStrategy_GetDefaultSecurityGroupName(t *testing.T) {
    // Test naming logic
}

func TestEC2Client_GetOrCreateSecurityGroup(t *testing.T) {
    // Test: SSH rules, Session Manager rules, creation
}
```

**Testing Strategy:**
1. **Table-driven tests** for logic/algorithms
2. **Mock AWS SDK** for integration points (use aws-sdk-go-v2 mock package)
3. **Error path testing** for all AWS API calls
4. **Validation testing** for all input parameters

**Estimated Time:** 16-20 hours
**Complexity:** High
**Risk:** Medium

---

### 2.2 Add Tests for Configuration (`internal/config/*`)

#### 2.2.1 User Data Generation (`internal/config/userdata.go` - 0%)
```go
// internal/config/userdata_test.go
func TestGenerateUserData(t *testing.T) {
    env := &Environment{
        Name: "test",
        Packages: []string{"python3", "git"},
        PipPackages: []string{"jupyter", "pandas"},
    }

    userData, err := GenerateUserData(env)
    require.NoError(t, err)

    // Decode base64
    decoded, err := base64.StdEncoding.DecodeString(userData)
    require.NoError(t, err)

    script := string(decoded)

    // Verify script structure
    assert.Contains(t, script, "#!/bin/bash")
    assert.Contains(t, script, "apt-get install -y")
    assert.Contains(t, script, "python3", "git")
    assert.Contains(t, script, "pip install")
    assert.Contains(t, script, "jupyter", "pandas")
    assert.Contains(t, script, "systemctl enable jupyter.service")
}

func TestGenerateUserData_EmptyEnvironment(t *testing.T) {
    // Test minimal environment
}

func TestGenerateUserData_ComplexEnvironment(t *testing.T) {
    // Test with extensions, env vars, etc.
}

func TestGenerateUserDataScript_ValidBash(t *testing.T) {
    // Validate generated bash is syntactically correct
    // Use `bash -n` to check syntax
}
```

#### 2.2.2 Key Storage (`internal/config/keys.go` - 0%)
```go
// internal/config/keys_test.go
func TestKeyStorage_SavePrivateKey(t *testing.T) {
    // Test: save new key, overwrite existing, permissions
}

func TestKeyStorage_LoadPrivateKey(t *testing.T) {
    // Test: load existing, not found, permission errors
}

func TestKeyStorage_ValidateKeyPermissions(t *testing.T) {
    // Test: correct 600, incorrect permissions
}

func TestKeyStorage_CleanupOrphanedKeys(t *testing.T) {
    // Test: cleanup logic, matching keys
}
```

**Estimated Time:** 6-8 hours
**Complexity:** Medium
**Risk:** Low

---

## Phase 3: CLI Testing & Coverage (Day 4) - HIGH PRIORITY

**Priority: P1 - User-facing code must be reliable**

### 3.1 Improve CLI Test Coverage (35.7% → 70%+)

#### 3.1.1 Launch Command (`internal/cli/launch.go`)
```go
// internal/cli/launch_test.go (expand existing)

func TestRunLaunch_DryRun_SSH(t *testing.T) {
    // Test dry run with SSH connection method
}

func TestRunLaunch_DryRun_SessionManager(t *testing.T) {
    // Test dry run with Session Manager
}

func TestRunLaunch_DryRun_PrivateSubnet(t *testing.T) {
    // Test dry run with private subnet + NAT Gateway
}

func TestRunLaunch_InvalidConnectionMethod(t *testing.T) {
    // Test error handling
}

func TestRunLaunch_InvalidSubnetType(t *testing.T) {
    // Test error handling
}
```

#### 3.1.2 Key Management Commands (`internal/cli/key.go` - 0%)
```go
// internal/cli/key_test.go (new file)

func TestRunKeyList(t *testing.T) {
    // Mock AWS API, test list output formatting
}

func TestRunKeyCleanup(t *testing.T) {
    // Test cleanup logic with mocked keys
}

func TestRunKeyValidate(t *testing.T) {
    // Test validation output
}

func TestRunKeyShow(t *testing.T) {
    // Test key detail display
}
```

#### 3.1.3 Instance Lifecycle Commands (0%)
```go
// internal/cli/stop_test.go (new file)
func TestRunStop(t *testing.T) {
    // Test stop logic
}

// internal/cli/terminate_test.go (new file)
func TestRunTerminate(t *testing.T) {
    // Test terminate with confirmation
}

// internal/cli/status_test.go (new file)
func TestRunStatus(t *testing.T) {
    // Test status display
}

// internal/cli/connect_test.go (new file)
func TestRunConnect(t *testing.T) {
    // Test connection setup
}
```

**Estimated Time:** 10-12 hours
**Complexity:** Medium
**Risk:** Low

---

### 3.2 Add Main Entry Point Tests (`cmd/aws-jupyter/main.go` - 0%)

```go
// cmd/aws-jupyter/main_test.go (new file)

func TestMain_VersionFlag(t *testing.T) {
    // Test --version output
}

func TestMain_HelpFlag(t *testing.T) {
    // Test --help output
}

func TestMain_CommandRouting(t *testing.T) {
    // Test each command is registered correctly
}
```

**Estimated Time:** 2-3 hours
**Complexity:** Low
**Risk:** Low

---

## Phase 4: E2E Testing (Day 5) - PRODUCTION VALIDATION

**Priority: P2 - Validates full workflows**

### 4.1 Add E2E Test Framework

```go
// test/e2e/e2e_test.go (new file)

// +build e2e

func TestE2E_LaunchSSH(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }

    // Requires: AWS credentials, real AWS account
    // 1. Launch instance with SSH
    // 2. Verify instance running
    // 3. Verify SSH key created
    // 4. Verify security group created
    // 5. Terminate instance
    // 6. Cleanup resources
}

func TestE2E_LaunchSessionManager(t *testing.T) {
    // Full workflow with Session Manager
}

func TestE2E_LaunchPrivateSubnet(t *testing.T) {
    // Full workflow with private subnet + NAT
}

func TestE2E_KeyPairReuse(t *testing.T) {
    // Verify key pair reuse across launches
}

func TestE2E_InstanceLifecycle(t *testing.T) {
    // Launch → Stop → Start → Terminate
}
```

### 4.2 E2E Test Infrastructure

```makefile
# Makefile additions
.PHONY: test-e2e
test-e2e:
	@echo "Running E2E tests (requires AWS credentials)..."
	AWS_PROFILE=test go test -v -tags=e2e -timeout=30m ./test/e2e/...

.PHONY: test-e2e-cleanup
test-e2e-cleanup:
	@echo "Cleaning up E2E test resources..."
	go run ./test/e2e/cleanup/main.go
```

```yaml
# .github/workflows/e2e.yml (new file)
name: E2E Tests

on:
  schedule:
    - cron: '0 2 * * *'  # Run nightly
  workflow_dispatch:     # Manual trigger

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Configure AWS Credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ secrets.AWS_E2E_ROLE }}
          aws-region: us-east-1
      - name: Run E2E Tests
        run: make test-e2e
      - name: Cleanup Resources
        if: always()
        run: make test-e2e-cleanup
```

**Estimated Time:** 8-10 hours
**Complexity:** High
**Risk:** Low (optional for initial release)

---

## Phase 5: Documentation & Polish (Day 6) - QUALITY

**Priority: P2 - Required for A+ rating**

### 5.1 Add Package Documentation

```go
// internal/aws/doc.go (new file)
/*
Package aws provides AWS service clients and utilities for managing
EC2 instances, IAM roles, VPC networking, and security groups.

This package handles all interactions with AWS APIs including:
  - EC2 instance lifecycle management
  - AMI selection and validation
  - IAM role and instance profile creation
  - SSH key pair management
  - VPC subnet and NAT gateway configuration
  - Security group management

Example usage:

    ctx := context.Background()
    client, err := aws.NewEC2Client(ctx, "default")
    if err != nil {
        log.Fatal(err)
    }

    params := aws.LaunchParams{
        AMI:          "ami-12345678",
        InstanceType: "t3.medium",
        KeyPairName:  "my-key",
    }

    instance, err := client.LaunchInstance(ctx, params)
    if err != nil {
        log.Fatal(err)
    }
*/
package aws

// internal/cli/doc.go (new file)
/*
Package cli implements the command-line interface for aws-jupyter.

It provides commands for launching, managing, and connecting to
Jupyter Lab instances on AWS EC2. The CLI is built using Cobra
and supports environment-based configuration.

Commands:
  - launch: Launch a new Jupyter instance
  - list: List running instances
  - stop: Stop an instance
  - terminate: Terminate an instance
  - connect: Connect to an instance
  - key: Manage SSH key pairs
  - env: Manage environment configurations
  - generate: Generate environment from local setup

Example:

    aws-jupyter launch --env data-science --instance-type m7g.large
*/
package cli

// internal/config/doc.go (new file)
/*
Package config handles configuration management for aws-jupyter.

This includes:
  - Environment configuration loading and validation
  - SSH key storage and permissions management
  - Instance state persistence
  - User data script generation

Configuration files are stored in ~/.aws-jupyter/ by default.
*/
package config
```

### 5.2 Add README.md Sections

```markdown
## Testing

### Running Tests

```bash
# Run all unit tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Run E2E tests (requires AWS credentials)
make test-e2e
```

### Code Quality

This project maintains high code quality standards:
- **Go Report Card:** A+ rating
- **Test Coverage:** 70%+ overall, 80%+ critical paths
- **Linting:** Zero golangci-lint issues
- **Cyclomatic Complexity:** All functions < 15

### Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.
```

### 5.3 Add CONTRIBUTING.md

```markdown
# Contributing to aws-jupyter

## Code Quality Standards

### Testing Requirements
- All new features must include unit tests
- Critical paths require 80%+ coverage
- Integration tests for AWS interactions
- E2E tests for user workflows

### Code Style
- Run `go fmt` before committing
- Run `go vet` to catch common issues
- Run `golangci-lint run` for comprehensive linting
- All exported symbols must have documentation comments
- Keep cyclomatic complexity < 15

### Pull Request Process
1. Fork the repository
2. Create a feature branch
3. Write tests for your changes
4. Ensure all tests pass
5. Run linters and fix issues
6. Update documentation
7. Submit pull request

See [.github/PULL_REQUEST_TEMPLATE.md] for checklist.
```

**Estimated Time:** 4-5 hours
**Complexity:** Low
**Risk:** None

---

## Phase 6: CI/CD Enhancements (Day 7) - AUTOMATION

**Priority: P3 - Prevents regression**

### 6.1 Enhance GitHub Actions Workflow

```yaml
# .github/workflows/ci.yml (enhance existing)

jobs:
  test:
    # ... existing ...

    - name: Run tests with coverage
      run: go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

    - name: Check coverage threshold
      run: |
        COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3, 1, length($3)-1)}')
        echo "Total coverage: $COVERAGE%"
        if (( $(echo "$COVERAGE < 70" | bc -l) )); then
          echo "❌ Coverage $COVERAGE% is below 70% threshold"
          exit 1
        fi
        echo "✅ Coverage $COVERAGE% meets threshold"

    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v4
      with:
        file: ./coverage.out
        flags: unittests
        fail_ci_if_error: true

  lint:
    # ... existing ...

    - name: Run gocyclo
      run: |
        go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
        gocyclo -over 15 . | tee gocyclo.txt
        if [ -s gocyclo.txt ]; then
          echo "❌ Functions with complexity > 15 found"
          cat gocyclo.txt
          exit 1
        fi

    - name: Run ineffassign
      run: |
        go install github.com/gordonklaus/ineffassign@latest
        ineffassign ./...

    - name: Run misspell
      run: |
        go install github.com/client9/misspell/cmd/misspell@latest
        misspell -error .

    - name: Go Report Card simulation
      run: |
        echo "Checking Go Report Card criteria..."
        ERRORS=0

        # gofmt
        if [ "$(gofmt -l . | wc -l)" -gt 0 ]; then
          echo "❌ gofmt issues found"
          ERRORS=$((ERRORS+1))
        fi

        # go vet
        if ! go vet ./...; then
          echo "❌ go vet issues found"
          ERRORS=$((ERRORS+1))
        fi

        # golangci-lint
        if ! golangci-lint run --timeout=5m; then
          echo "❌ golangci-lint issues found"
          ERRORS=$((ERRORS+1))
        fi

        if [ $ERRORS -gt 0 ]; then
          echo "❌ Go Report Card simulation failed with $ERRORS error(s)"
          exit 1
        fi

        echo "✅ Go Report Card simulation passed"

  quality-gate:
    needs: [test, lint, build]
    runs-on: ubuntu-latest
    steps:
      - name: Quality gate passed
        run: echo "✅ All quality checks passed"
```

### 6.2 Add Pre-commit Hooks

```bash
# .githooks/pre-commit (new file)
#!/bin/bash

echo "Running pre-commit checks..."

# Format check
if [ "$(gofmt -l . | wc -l)" -gt 0 ]; then
    echo "❌ Code is not formatted. Run: go fmt ./..."
    gofmt -l .
    exit 1
fi

# Vet check
if ! go vet ./... 2>&1; then
    echo "❌ go vet failed"
    exit 1
fi

# Test check
if ! go test ./... -short 2>&1; then
    echo "❌ Tests failed"
    exit 1
fi

echo "✅ Pre-commit checks passed"
```

```bash
# Install hooks
git config core.hooksPath .githooks
chmod +x .githooks/pre-commit
```

### 6.3 Add Makefile for Common Tasks

```makefile
# Makefile (new file)
.PHONY: all test lint fmt vet build install clean coverage report-card help

# Default target
all: lint test build

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -cover ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linters
lint:
	@echo "Running linters..."
	go fmt ./...
	go vet ./...
	golangci-lint run --timeout=5m
	gocyclo -over 15 .
	ineffassign ./...
	misspell -error .

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	go vet ./...

# Build binary
build:
	@echo "Building..."
	go build -v -o bin/aws-jupyter ./cmd/aws-jupyter

# Install binary
install:
	@echo "Installing..."
	go install ./cmd/aws-jupyter

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/ coverage.out coverage.html

# Simulate Go Report Card
report-card: lint test
	@echo "✅ Go Report Card simulation complete"

# Install development tools
tools:
	@echo "Installing development tools..."
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	go install github.com/client9/misspell/cmd/misspell@latest
	go install github.com/gordonklaus/ineffassign@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Show help
help:
	@echo "Available targets:"
	@echo "  all         - Run lint, test, and build (default)"
	@echo "  test        - Run unit tests"
	@echo "  coverage    - Generate coverage report"
	@echo "  lint        - Run all linters"
	@echo "  fmt         - Format code"
	@echo "  vet         - Run go vet"
	@echo "  build       - Build binary"
	@echo "  install     - Install binary"
	@echo "  clean       - Remove build artifacts"
	@echo "  report-card - Simulate Go Report Card checks"
	@echo "  tools       - Install development tools"
	@echo "  help        - Show this help"
```

**Estimated Time:** 3-4 hours
**Complexity:** Low
**Risk:** None

---

## Implementation Checklist

### Phase 1: Critical Fixes (Day 1)
- [ ] Fix 16 errcheck violations in production code
- [ ] Fix 5 errcheck violations in test code
- [ ] Fix 1 ineffassign violation in ami.go
- [ ] Refactor runLaunch (complexity 49 → <15)
  - [ ] Create loadAndValidateEnvironment
  - [ ] Create executeDryRun
  - [ ] Create executeLaunch with sub-functions
  - [ ] Create setupAWSClient
  - [ ] Create setupAuthentication
  - [ ] Create setupNetworking
  - [ ] Create setupSecurity
  - [ ] Create launchAndDisplayInstance
  - [ ] Update all tests
- [ ] Add documentation to 30+ exported symbols
- [ ] Fix all ID suffix violations (Id → ID)
- [ ] Run full linting suite (0 issues)
- [ ] Commit: "fix: resolve all linting issues for Go Report Card A+"

### Phase 2: AWS Infrastructure Testing (Days 2-3)
- [ ] Create internal/aws/ami_test.go (6 tests)
- [ ] Create internal/aws/iam_test.go (8 tests)
- [ ] Create internal/aws/keypair_test.go (10 tests)
- [ ] Create internal/aws/networking_test.go (8 tests)
- [ ] Create internal/aws/security_test.go (10 tests)
- [ ] Create internal/config/userdata_test.go (8 tests)
- [ ] Create internal/config/keys_test.go (10 tests)
- [ ] Verify aws package coverage >60%
- [ ] Verify config package coverage >60%
- [ ] Commit: "test: add comprehensive AWS infrastructure tests"

### Phase 3: CLI Testing & Coverage (Day 4)
- [ ] Expand internal/cli/launch_test.go (5 new tests)
- [ ] Create internal/cli/key_test.go (10 tests)
- [ ] Create internal/cli/stop_test.go (4 tests)
- [ ] Create internal/cli/terminate_test.go (4 tests)
- [ ] Create internal/cli/status_test.go (4 tests)
- [ ] Create internal/cli/connect_test.go (4 tests)
- [ ] Create cmd/aws-jupyter/main_test.go (3 tests)
- [ ] Verify cli package coverage >70%
- [ ] Verify overall coverage >70%
- [ ] Commit: "test: achieve 70% test coverage across all packages"

### Phase 4: E2E Testing (Day 5)
- [ ] Create test/e2e directory structure
- [ ] Create test/e2e/e2e_test.go (5 E2E tests)
- [ ] Create test/e2e/cleanup tool
- [ ] Create .github/workflows/e2e.yml
- [ ] Update Makefile with e2e targets
- [ ] Test E2E suite in real AWS account
- [ ] Commit: "test: add E2E test suite for production validation"

### Phase 6: Documentation & Polish (Day 6)
- [ ] Create internal/aws/doc.go
- [ ] Create internal/cli/doc.go
- [ ] Create internal/config/doc.go
- [ ] Update README.md with testing section
- [ ] Create CONTRIBUTING.md
- [ ] Add code examples to key functions
- [ ] Review all godoc output
- [ ] Commit: "docs: add comprehensive package and contribution documentation"

### Phase 6: CI/CD Enhancements (Day 7)
- [ ] Enhance .github/workflows/ci.yml
- [ ] Add coverage threshold check (70%)
- [ ] Add Go Report Card simulation
- [ ] Create .githooks/pre-commit
- [ ] Create Makefile
- [ ] Update README with make commands
- [ ] Test full CI pipeline
- [ ] Commit: "ci: enhance CI/CD with quality gates and automation"

### Final Validation
- [ ] Run `make report-card` (0 issues)
- [ ] Run `make coverage` (>70%)
- [ ] Run `make test-e2e` (all passing)
- [ ] Submit to goreportcard.com (verify A+)
- [ ] Tag release v0.2.0
- [ ] Update CHANGELOG.md

---

## Success Metrics

**Code Quality (Go Report Card):**
- ✅ gofmt: 100%
- ✅ go vet: 100%
- ✅ gocyclo: 100% (all functions <15)
- ✅ golint: 100%
- ✅ ineffassign: 100%
- ✅ misspell: 100%
- ✅ license: 100%
- 🎯 **Overall: A+ (99%+)**

**Test Coverage:**
- cmd/aws-jupyter: 0% → 80%+
- internal/aws: 0.9% → 70%+
- internal/cli: 35.7% → 70%+
- internal/config: 19.2% → 70%+
- 🎯 **Overall: 19.9% → 70%+**

**Production Readiness:**
- ✅ Zero linting issues
- ✅ All error returns checked
- ✅ All functions documented
- ✅ E2E tests passing
- ✅ CI quality gates enforced
- 🎯 **Ready for v0.2.0 release**

---

## Risk Mitigation

**Refactoring Risk (runLaunch complexity):**
- ⚠️ Medium risk - extensive refactoring
- ✅ Mitigation: Comprehensive test coverage before refactoring
- ✅ Mitigation: Incremental commits with verification
- ✅ Mitigation: Keep existing tests passing throughout

**AWS Testing Risk:**
- ⚠️ Testing real AWS services requires credentials
- ✅ Mitigation: Use AWS SDK mocking for unit tests
- ✅ Mitigation: E2E tests in isolated test account
- ✅ Mitigation: Automatic cleanup of test resources

**Timeline Risk:**
- ⚠️ 40-50 hours estimated (6-7 days)
- ✅ Mitigation: Phases are independent, can be staged
- ✅ Mitigation: Phase 1 (critical) can be completed in 1 day
- ✅ Mitigation: Phase 4 (E2E) is optional for initial release

---

## Notes

**Priority Order:**
1. **Phase 1 (Day 1)** - Blocking, required for A+ rating
2. **Phase 2-3 (Days 2-4)** - Critical for production
3. **Phase 4 (Day 5)** - Important for validation
4. **Phase 5-6 (Days 6-7)** - Quality and automation

**Quick Win Path (2 days):**
If time is constrained, focus on:
- Day 1: Complete Phase 1 (critical fixes)
- Day 2: Complete 2.1.3, 2.2.1, 3.1.1 (key pair, userdata, launch tests)
- Result: A+ rating + 50% coverage + production-ready core features

**Dependencies:**
- Go 1.22+
- golangci-lint
- gocyclo, misspell, ineffassign
- AWS SDK Go v2
- AWS test account (for E2E tests)

---

**Status:** Ready for implementation
**Approval Required:** Yes
**Estimated Completion:** 5-7 days (40-50 hours)
**Target Release:** v0.2.0
