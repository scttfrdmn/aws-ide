# Contributing to aws-jupyter

🎉 **Thank you for your interest in contributing to aws-jupyter!**

This project is currently in active development, and we welcome contributions of all kinds - from bug reports and feature requests to code contributions and documentation improvements.

## 📋 Table of Contents

- [Quick Start](#quick-start)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Contributing Workflow](#contributing-workflow)
- [Code Standards](#code-standards)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Getting Help](#getting-help)

## 🚀 Quick Start

1. **Fork the repository** on GitHub
2. **Clone your fork**: `git clone git@github.com:YOUR_USERNAME/aws-jupyter.git`
3. **Install dependencies**: `go mod tidy`
4. **Run tests**: `go test ./...`
5. **Build the project**: `go build -o aws-jupyter cmd/aws-jupyter/main.go`

## 🛠 Development Setup

### Prerequisites

- **Go 1.21+** - [Download here](https://golang.org/dl/)
- **Git** - Version control
- **AWS CLI** (optional) - For testing AWS integration
- **Pre-commit** (optional) - `pip install pre-commit && pre-commit install`

### Environment Setup

```bash
# Clone the repository
git clone git@github.com:scttfrdmn/aws-jupyter.git
cd aws-jupyter

# Install dependencies
go mod tidy

# Install development tools (optional but recommended)
go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
go install github.com/kisielk/errcheck@latest
go install github.com/client9/misspell/cmd/misspell@latest

# Install pre-commit hooks (optional)
pip install pre-commit
pre-commit install

# Build and test
go build -o aws-jupyter cmd/aws-jupyter/main.go
go test ./...
```

## 📁 Project Structure

```
aws-jupyter/
├── cmd/aws-jupyter/        # Main CLI entry point
├── internal/
│   ├── aws/                # AWS EC2 client and operations
│   ├── cli/                # CLI commands (launch, list, etc.)
│   └── config/             # Configuration and state management
├── environments/           # Built-in environment templates
├── docs/                   # Documentation
├── .github/                # GitHub templates and workflows
└── tests/                  # Integration tests (planned)
```

## 🔄 Contributing Workflow

### 1. Choose What to Work On

- **Check existing issues** - Look for `good first issue` or `help wanted` labels
- **Review the roadmap** - See what features are planned in the README
- **Propose new features** - Open an issue to discuss before implementing

### 2. Development Process

```bash
# Create a feature branch
git checkout -b feature/your-feature-name

# Make your changes
# Write tests for new functionality
# Update documentation as needed

# Test your changes
go test ./...
go vet ./...
gofmt -w .

# Commit with conventional commits
git commit -m "feat: add SSH key pair management"

# Push and create pull request
git push origin feature/your-feature-name
```

### 3. Areas Needing Help

**🔥 High Priority:**
- SSH key pair management (`internal/aws/`)
- Security group setup (`internal/aws/`)
- User data script generation (`internal/cli/`)
- EC2 instance launching (`internal/aws/`)

**📚 Documentation:**
- AWS authentication guide (`docs/AWS_AUTHENTICATION.md`)
- Usage examples and tutorials
- API documentation

**🧪 Testing:**
- Integration tests for AWS operations
- End-to-end testing scenarios
- Mock AWS responses for unit tests

## ✅ Code Standards

We maintain **A+ Go Report Card** standards:

### Code Quality Requirements

- **Formatting**: `gofmt -w .` (enforced by pre-commit)
- **Linting**: All `go vet` issues must be resolved
- **Complexity**: Functions should have cyclomatic complexity ≤15
- **Error Handling**: All errors must be properly handled
- **Testing**: New code should include unit tests

### Code Style Guidelines

```go
// ✅ Good: Clear function names and error handling
func CreateKeyPair(ctx context.Context, name string) (*ec2.KeyPair, error) {
    if name == "" {
        return nil, fmt.Errorf("key pair name cannot be empty")
    }

    result, err := e.client.CreateKeyPair(ctx, &ec2.CreateKeyPairInput{
        KeyName: aws.String(name),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create key pair: %w", err)
    }

    return result.KeyPair, nil
}

// ❌ Avoid: Unchecked errors, unclear naming
func CreateKey(name string) *ec2.KeyPair {
    result, _ := client.CreateKeyPair(nil, &ec2.CreateKeyPairInput{
        KeyName: &name,
    })
    return result.KeyPair
}
```

### Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

```bash
feat: add SSH key pair management
fix: resolve authentication timeout issue
docs: update AWS authentication guide
test: add unit tests for environment loading
refactor: simplify EC2 client creation
```

## 🧪 Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run specific package tests
go test ./internal/cli -v

# Run quality checks
go vet ./...
gocyclo -over 15 .
errcheck ./...
```

### Test Coverage Requirements

- **Core packages** (`internal/cli`, `internal/config`): >70% coverage
- **AWS operations** (`internal/aws`): Best effort (limited by AWS mocking)
- **New functionality**: Should include comprehensive unit tests

### Writing Tests

```go
func TestCreateKeyPair(t *testing.T) {
    // Test data setup
    ctx := context.Background()
    keyName := "test-key-" + uuid.New().String()

    // Test execution
    keyPair, err := client.CreateKeyPair(ctx, keyName)

    // Assertions
    if err != nil {
        t.Fatalf("CreateKeyPair failed: %v", err)
    }
    if keyPair.KeyName == nil || *keyPair.KeyName != keyName {
        t.Errorf("Expected key name %s, got %v", keyName, keyPair.KeyName)
    }
}
```

## 📥 Submitting Changes

### Pull Request Process

1. **Ensure tests pass** - All CI checks must be green
2. **Update documentation** - Include relevant docs updates
3. **Add changelog entry** - Update `CHANGELOG.md` if needed
4. **Request review** - Tag maintainers for review
5. **Address feedback** - Respond to review comments promptly

### Pull Request Template

When you create a PR, please include:

- **What changed** - Brief description of your changes
- **Why** - Context and motivation for the change
- **Testing** - How you tested your changes
- **Documentation** - Any docs updates needed
- **Breaking changes** - Note any backwards compatibility issues

## 🤝 Getting Help

### Communication Channels

- **GitHub Issues** - Bug reports, feature requests, questions
- **GitHub Discussions** - General questions and community chat
- **Pull Request Reviews** - Code-specific questions and feedback

### Maintainer Response Times

- **Issues**: We aim to respond within 2-3 business days
- **Pull Requests**: Initial review within 3-5 business days
- **Security Issues**: Please email privately for faster response

### Development Questions

Common questions and answers:

**Q: How do I test AWS operations without real AWS resources?**
A: We're working on AWS mocking. For now, use `--dry-run` mode and unit test the logic around AWS calls.

**Q: What AWS permissions does aws-jupyter need?**
A: See the [AWS Authentication Guide](docs/AWS_AUTHENTICATION.md) for detailed permission requirements.

**Q: How do I add a new CLI command?**
A: Look at existing commands in `internal/cli/` for patterns. Each command needs a `New*Cmd()` function and corresponding tests.

## 📝 Development Notes

### Current Implementation Status

- ✅ **CLI Framework**: Complete with all command structures
- ✅ **Environment System**: 6 built-in environments, custom environment support
- ✅ **AWS Authentication**: Full credential chain support
- 🚧 **AWS Operations**: Key pair and security group management in progress
- ❌ **Instance Management**: Launch, stop, terminate not yet implemented
- ❌ **SSH Tunneling**: Port forwarding not yet implemented

### Architecture Decisions

- **No external dependencies** for core functionality where possible
- **AWS SDK v2** for all AWS operations
- **Cobra** for CLI framework (industry standard)
- **Local state management** using JSON files in `~/.aws-jupyter/`
- **YAML configuration** for environments (human-readable)

---

## 💡 Thank You!

Your contributions make aws-jupyter better for everyone. Whether you're fixing typos, adding features, or helping other contributors, your efforts are appreciated! 🙏

**Questions?** Don't hesitate to open an issue or start a discussion. We're here to help!