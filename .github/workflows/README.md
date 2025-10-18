# GitHub Actions CI/CD Workflows

This directory contains the CI/CD workflows for the aws-ide project.

## Workflows Overview

### 1. CI (`ci.yml`) - Automatic on Push/PR

**Trigger:** Runs automatically on every push to `main` and all pull requests

**Jobs:**
- **Test**: Unit tests with race detection across Go 1.22 and 1.23
- **Lint**: Code quality checks (gofmt, go vet, golangci-lint)
- **Integration**: Integration tests with LocalStack (mocked AWS services)
- **Build**: Build all three applications and verify they work

**Duration:** ~5-10 minutes
**Cost:** Free (GitHub-hosted runners)
**AWS Resources:** None (uses LocalStack for AWS API mocking)

This is the primary quality gate for all code changes.

### 2. Smoke Tests (`smoke-tests.yml`) - Manual Trigger

**Trigger:** Manual workflow dispatch only

**Purpose:** Quick validation against real AWS infrastructure

**What it tests:**
- AWS credential configuration
- Real AWS API connectivity
- Instance type availability
- AMI discovery
- IAM role verification
- Subnet discovery

**Duration:** ~5-10 minutes
**Cost:** ~$0.02-0.05 per run
**AWS Resources:** Minimal (describe-only operations, no instance launches)

**When to run:**
- Before releases to verify AWS integration
- After major infrastructure changes
- When troubleshooting AWS-specific issues

**Configuration:**
- Requires `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` secrets
- Allows selection of AWS region for testing

### 3. E2E Tests (`e2e-tests.yml`) - Manual Trigger

**Trigger:** Manual workflow dispatch only

**Purpose:** Full end-to-end testing of complete workflows

**What it tests:**
- Complete launch → connect → terminate flows
- All three IDE types (Jupyter, RStudio, VSCode)
- Session Manager and SSH connectivity
- Environment customization
- Multi-architecture support (ARM64/x86)
- Instance lifecycle operations

**Duration:** ~20-40 minutes
**Cost:** ~$0.06-0.20 per suite
**AWS Resources:** Creates real EC2 instances, security groups, IAM roles (auto-cleanup)

**When to run:**
- Before major releases
- After significant feature changes
- When validating fixes for user-reported issues
- **NOT** on every commit (too slow and costly)

**Configuration:**
- Requires `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` secrets
- Allows selection of AWS region
- Allows selection of specific test suite (all/jupyter/rstudio/vscode)
- Includes automatic cleanup of test resources
- Provides cost estimate in workflow summary

### 4. Release (`release.yml`) - Automatic on Tags

**Trigger:** Runs automatically when a git tag matching `v*` is pushed

**Purpose:** Build and publish release binaries

**What it does:**
- Builds cross-platform binaries for all three applications
- Creates GitHub release with release notes
- Uploads binaries as release artifacts
- Supports 10 OS/architecture combinations

**Platforms:**
- Linux: amd64, arm64
- macOS: amd64, arm64
- Windows: amd64, arm64

**Duration:** ~10-15 minutes
**Cost:** Free (GitHub-hosted runners)

## Testing Strategy

### Test Pyramid

```
        /\
       /E2E\        <- Manual, slow, real AWS, high cost
      /------\
     / Smoke  \     <- Manual, fast, real AWS, low cost
    /----------\
   /Integration\   <- Auto, fast, mocked AWS, no cost
  /--------------\
 /   Unit Tests   \ <- Auto, fastest, no AWS, no cost
/------------------\
```

### When Each Test Type Runs

| Test Type | Trigger | AWS | Cost | Duration | Purpose |
|-----------|---------|-----|------|----------|---------|
| Unit | Every commit | Mocked | $0 | 1-2 min | Fast feedback on logic |
| Integration | Every commit | LocalStack | $0 | 3-5 min | AWS SDK integration |
| Lint | Every commit | None | $0 | 2-3 min | Code quality |
| Build | Every commit | None | $0 | 1-2 min | Binary compilation |
| Smoke | Manual | Real | ~$0.05 | 5-10 min | Pre-release validation |
| E2E | Manual | Real | ~$0.20 | 20-40 min | Complete workflow testing |

### Test Coverage Goals (v1.0.0)

- Unit tests: 60%+ coverage
- Integration tests: All AWS operations
- Smoke tests: All critical AWS checks
- E2E tests: All user workflows for all IDE types

## Running Workflows Manually

### Smoke Tests

1. Go to **Actions** → **Smoke Tests**
2. Click **Run workflow**
3. Select AWS region
4. Click **Run workflow**

### E2E Tests

1. Go to **Actions** → **E2E Tests**
2. Click **Run workflow**
3. Select AWS region
4. Select test suite (all/jupyter/rstudio/vscode)
5. Click **Run workflow**
6. Review cost estimate in workflow summary after completion

## Required Secrets

To run smoke and E2E tests, configure these repository secrets:

- `AWS_ACCESS_KEY_ID`: AWS access key with EC2, IAM, SSM permissions
- `AWS_SECRET_ACCESS_KEY`: Corresponding secret key

**Permissions required:**
- ec2:* (instance management)
- iam:CreateRole, iam:AttachRolePolicy, iam:GetRole, etc.
- ssm:SendCommand, ssm:GetCommandInvocation

## Local Testing

You can run the same tests locally using the `Makefile`:

```bash
# Unit tests (fast, no AWS)
make test-unit

# Integration tests (requires LocalStack)
docker-compose up -d  # Start LocalStack
make test-integration

# Smoke tests (requires AWS credentials)
make test-smoke

# E2E tests (requires AWS credentials, costs money)
make test-e2e
```

## Workflow Maintenance

### Updating Go Version

Update the Go version in all workflow files when upgrading:
- `ci.yml`: Update `matrix.go` and `go-version`
- `smoke-tests.yml`: Update `go-version`
- `e2e-tests.yml`: Update `go-version`
- `release.yml`: Update `go-version`

### Adding New Applications

When adding a new application to `apps/`:

1. Add to `ci.yml` matrix: `module` and `app`
2. Add to `smoke-tests.yml`: new test step
3. Add to `e2e-tests.yml`: new test step and option
4. Add to `release.yml`: new goreleaser config

### Optimizing Costs

E2E tests are the primary cost driver. To minimize costs:

1. **Run selectively**: Only run full E2E suite before releases
2. **Use specific suites**: Test only the changed IDE type
3. **Choose cheaper regions**: us-east-1 is typically cheapest
4. **Cleanup verified**: Workflows include automatic cleanup
5. **Monitor usage**: Check workflow summaries for cost estimates

## Troubleshooting

### LocalStack Integration Tests Failing

If integration tests fail with connection errors:
1. Check LocalStack service health in workflow logs
2. Verify the health check is passing
3. Increase wait timeout if needed (line 184 in ci.yml)

### Smoke/E2E Tests Can't Connect to AWS

If AWS tests fail with credential errors:
1. Verify secrets are configured correctly
2. Check IAM permissions are sufficient
3. Verify the AWS region supports your instance types

### E2E Tests Leaving Resources

If cleanup fails:
1. Manually check for instances with tag `CreatedBy=aws-ide-test`
2. Terminate orphaned instances
3. Delete associated security groups and IAM roles
4. Review cleanup logic in e2e-tests.yml

## Cost Monitoring

Track CI/CD costs in the repository:

- **Unit/Integration/Lint/Build**: $0 (included in GitHub Actions free tier)
- **Smoke tests**: ~$0.50-1.00/month (if run weekly)
- **E2E tests**: ~$1-2/month (if run on releases only)

**Total estimated monthly cost: $1.50-3.00**

## Further Reading

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Testing Strategy](../../TESTING.md)
- [LocalStack Documentation](https://docs.localstack.cloud/)
- [GoReleaser Documentation](https://goreleaser.com/)
