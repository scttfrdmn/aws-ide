# Versioning Strategy

AWS IDE uses **independent versioning** for each application in the monorepo. This allows us to release updates to individual apps without unnecessarily bumping versions for unchanged apps.

## Version Format

Each app follows [Semantic Versioning](https://semver.org/):
- **MAJOR**: Incompatible API changes
- **MINOR**: New functionality (backwards compatible)
- **PATCH**: Bug fixes (backwards compatible)

## Git Tag Format

We use app-prefixed Git tags:

```
<app>/<version>
```

### Examples:
- `jupyter/v0.5.0` - Jupyter Lab launcher version 0.5.0
- `vscode/v0.1.0` - VSCode Server launcher version 0.1.0
- `rstudio/v0.5.0` - RStudio Server launcher version 0.5.0

## Current Versions

| App | Version | Status |
|-----|---------|--------|
| aws-jupyter | 0.5.0 | Stable |
| aws-vscode | 0.1.0 | Alpha |
| aws-rstudio | 0.5.0 | Beta |

## When to Release

### App-Specific Changes
If you modify **only one app** (files in `apps/<name>/`):
- Tag and release only that app
- Other apps remain at their current versions

**Example:**
```bash
# Fix bug in aws-vscode only
git tag vscode/v0.1.1
git push origin vscode/v0.1.1
```

### Shared Infrastructure Changes
If you modify **shared code** (`pkg/` module) that affects multiple apps:
- Consider which apps are actually impacted
- Release only the affected apps
- Coordinate version bumps if needed

**Example:**
```bash
# Fix IAM propagation bug in pkg/aws/ec2.go
# All apps use this, so release all three
git tag jupyter/v0.5.1
git tag vscode/v0.1.1
git tag rstudio/v0.5.1
git push origin --tags
```

### Breaking Changes in Shared Code
If you make breaking changes to `pkg/`:
- Bump MAJOR version for all affected apps
- Update all app code to work with new API
- Release all apps together

## Release Process

### 1. Update Version in Code
Edit `apps/<name>/cmd/aws-<name>/main.go`:
```go
var (
    version = "0.5.1"  // Update this
    commit  = "unknown"
    date    = "unknown"
)
```

### 2. Update CHANGELOG.md
Document changes in the `[Unreleased]` section.

### 3. Create Git Tag
```bash
git tag <app>/v<version>
git push origin <app>/v<version>
```

### 4. GitHub Actions
The release workflow automatically:
- Detects which app from the tag prefix
- Runs GoReleaser in the correct `apps/<name>/` directory
- Builds cross-platform binaries
- Creates GitHub release
- Updates Homebrew tap

## GoReleaser Configuration

Each app has its own `.goreleaser.yaml` in `apps/<name>/`:
- `apps/jupyter/.goreleaser.yaml`
- `apps/vscode/.goreleaser.yaml`
- `apps/rstudio/.goreleaser.yaml`

The root `.goreleaser.yaml` is **deprecated** and should not be used.

## Benefits

### Independent Releases
- Fix aws-vscode bugs without releasing jupyter/rstudio
- Users only update what they need
- Clearer release history per app

### Clear Changelogs
- Each release note shows exactly what app changed
- Easier to track which features are in which version
- Better user experience

### Flexible Development
- Work on experimental features in one app
- Keep stable apps at stable versions
- Alpha/Beta/Stable can coexist

## Migration from Shared Versioning

**Previous approach (v0.5.0 and earlier):**
- Single Git tag (`v0.5.0`)
- All apps released together
- Root `.goreleaser.yaml`

**New approach (v0.6.0+):**
- App-prefixed tags (`jupyter/v0.5.1`)
- Independent app releases
- Per-app `.goreleaser.yaml`

The old root `.goreleaser.yaml` remains for reference but should not be used for new releases.

## Examples

### Example 1: Bug Fix in aws-vscode Only

```bash
# Make changes to apps/vscode/
vim apps/vscode/internal/config/userdata.go

# Update version
vim apps/vscode/cmd/aws-vscode/main.go  # 0.1.0 → 0.1.1

# Update changelog
vim CHANGELOG.md

# Commit and tag
git add .
git commit -m "fix(vscode): set HOME environment variable for code-server"
git tag vscode/v0.1.1
git push origin main vscode/v0.1.1
```

**Result:** Only aws-vscode v0.1.1 is released. aws-jupyter and aws-rstudio remain at their current versions.

### Example 2: New Feature in Shared pkg/

```bash
# Add automatic retry logic to pkg/aws/ec2.go
vim pkg/aws/ec2.go

# Test with all apps
cd apps/jupyter && go test ./...
cd ../vscode && go test ./...
cd ../rstudio && go test ./...

# Update versions for all apps
vim apps/jupyter/cmd/aws-jupyter/main.go  # 0.5.0 → 0.5.1
vim apps/vscode/cmd/aws-vscode/main.go    # 0.1.0 → 0.1.1
vim apps/rstudio/cmd/aws-rstudio/main.go  # 0.5.0 → 0.5.1

# Update changelog
vim CHANGELOG.md

# Commit and tag all apps
git add .
git commit -m "fix: add automatic retry logic for IAM propagation delays"
git tag jupyter/v0.5.1
git tag vscode/v0.1.1
git tag rstudio/v0.5.1
git push origin main --tags
```

**Result:** All three apps get new releases with the shared infrastructure fix.

### Example 3: Major aws-jupyter Update

```bash
# Complete rewrite of jupyter environments system
vim apps/jupyter/internal/config/environment.go

# Breaking change: bump MAJOR version
vim apps/jupyter/cmd/aws-jupyter/main.go  # 0.5.1 → 1.0.0

# Update changelog
vim CHANGELOG.md

# Commit and tag
git add .
git commit -m "feat(jupyter)!: redesign environment configuration system"
git tag jupyter/v1.0.0
git push origin main jupyter/v1.0.0
```

**Result:** Only aws-jupyter gets v1.0.0. Other apps unaffected.

## Questions?

If you're unsure which apps to release:
1. Identify which files changed (`git diff`)
2. If only `apps/<name>/` changed → release that app only
3. If `pkg/` changed → test all apps and release affected ones
4. When in doubt, release all apps (safe but creates more releases)

## See Also

- [CHANGELOG.md](CHANGELOG.md) - Detailed release notes
- [ROADMAP.md](ROADMAP.md) - Future planning
- [Semantic Versioning](https://semver.org/) - Version number meanings
