# Versioning Strategy

AWS IDE uses a **dual versioning system** to track both platform stability and application-specific features independently.

## Overview

```
App Version: v0.6.0 (platform: v1.0.0)
             ^^^^^            ^^^^^
             │                └─ Platform/Infrastructure Version
             └─ Application Version
```

- **Platform Version** (`pkg/` module): Tracks shared infrastructure API stability
- **App Version** (individual apps): Tracks application-specific features

## Current Versions

| Component | Version | Status |
|-----------|---------|--------|
| Platform (`pkg/`) | v1.0.0 | Stable |
| aws-jupyter | v0.6.0 | Stable |
| aws-rstudio | v0.6.0 | Stable |
| aws-vscode | v0.6.0 | Beta |

## Platform Version (`pkg/` module)

**Location**: `pkg/version.go`
**Current Version**: `v1.0.0`
**Git Tag Format**: `pkg/v1.0.0`

### Semantic Versioning Rules

- **MAJOR** (`X.0.0`): Breaking changes to `pkg/` APIs
  - Changed function signatures
  - Removed public functions/types
  - Modified struct fields (breaking)

- **MINOR** (`1.X.0`): New features (backward compatible)
  - New public functions/types
  - New packages under `pkg/`

- **PATCH** (`1.0.X`): Bug fixes (backward compatible)
  - Internal bug fixes
  - Performance improvements

### Platform v1.0.0 Stable APIs

- `pkg/aws/ec2.go`: EC2 client and instance management
- `pkg/aws/iam.go`: IAM role and instance profile management
- `pkg/aws/networking.go`: VPC, subnet, security group management
- `pkg/aws/ssm.go`: SSM client and service readiness
- `pkg/config/state.go`: Instance state management
- `pkg/config/environment.go`: Environment configuration
- `pkg/config/keys.go`: SSH key management

## Application Versions

**Current**: All apps at `v0.6.0`
**Git Tag Format**: `v0.6.0` (unified) or `jupyter/v0.7.0` (app-specific)

### Semantic Versioning Rules

- **MAJOR** (`X.0.0`): Breaking CLI changes
  - Removed commands or flags
  - Changed command syntax

- **MINOR** (`0.X.0`): New features (backward compatible)
  - New commands
  - New flags
  - New environments

- **PATCH** (`0.6.X`): Bug fixes
  - Bug fixes
  - Documentation updates

### Unified vs Independent Versioning

**Currently**: Unified versioning (`v0.6.0` for all apps)

**Future option**: Independent versioning if apps diverge significantly
```
aws-jupyter@0.8.0   (got new features)
aws-rstudio@0.6.1   (only bug fixes)
aws-vscode@0.7.0    (got new features)
```

## Git Tag Strategy

### Current Tags
- **`v0.6.0`**: Unified release (all apps at v0.6.0, platform v1.0.0)
- **`pkg/v1.0.0`**: Platform version

### Future Options

**Option 1: Continue unified** (current)
```bash
git tag v0.7.0       # All apps at v0.7.0
git tag pkg/v1.1.0   # Platform gets new features
```

**Option 2: Independent apps** (if divergence occurs)
```bash
git tag jupyter/v0.8.0   # Only Jupyter updated
git tag rstudio/v0.6.1   # Only RStudio patched
```

## Version Check

### Check App Version
```bash
aws-jupyter --version
# Output: aws-jupyter version v0.6.0 (platform: v1.0.0, ...)
```

### Check Platform Version (in code)
```go
import "github.com/scttfrdmn/aws-ide/pkg"

fmt.Printf("Platform: %s\n", pkg.Version)
```

## Release Workflows

### Regular Feature Release (Unified)
1. Implement features
2. Update app versions in `apps/*/cmd/*/main.go`
3. Update CHANGELOG.md
4. Create tag: `git tag v0.7.0`
5. Push: `git push origin v0.7.0`

### Platform Breaking Change
1. Make breaking changes to `pkg/`
2. Update `pkg/version.go`: bump MAJOR (`1.0.0` → `2.0.0`)
3. Update all apps to work with new API
4. Bump app versions (likely MAJOR)
5. Create tags:
   ```bash
   git tag pkg/v2.0.0
   git tag v1.0.0
   git push origin pkg/v2.0.0 v1.0.0
   ```

### Platform Feature (Backward Compatible)
1. Add features to `pkg/`
2. Update `pkg/version.go`: bump MINOR (`1.0.0` → `1.1.0`)
3. Create tag: `git tag pkg/v1.1.0`
4. Apps adopt new features in future releases

## Version Compatibility

| Platform Version | Compatible App Versions |
|-----------------|------------------------|
| `pkg/v1.0.0`    | `v0.6.0+`              |
| `pkg/v1.1.0`    | `v0.6.0+`, `v0.7.0+`   |
| `pkg/v2.0.0`    | `v1.0.0+`              |

## FAQ

### Why dual versioning?

- **Clarity**: Distinguish platform stability from app features
- **Flexibility**: Apps can evolve while sharing stable infrastructure
- **Maintenance**: Platform can be versioned independently

### When to split app versions?

**Keep unified while**:
- Apps are released together
- Features span multiple apps
- Simplicity is valued

**Split when**:
- Apps diverge significantly
- Different release cadences emerge
- Users want stability in one app while another experiments

### How do I know which platform version an app requires?

```bash
aws-jupyter --version
# aws-jupyter version v0.6.0 (platform: v1.0.0, ...)
```

The `platform: v1.0.0` shows the required platform version.

## See Also

- [CHANGELOG.md](CHANGELOG.md) - Detailed release notes
- [ROADMAP.md](ROADMAP.md) - Future planning
- [Semantic Versioning](https://semver.org/) - Version number meanings
