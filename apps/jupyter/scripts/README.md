# Scripts

This directory contains utility scripts for maintaining the aws-jupyter project.

## update-homebrew-formula.sh

Automates updating the Homebrew formula in the tap repository after a new release.

**Usage:**
```bash
./scripts/update-homebrew-formula.sh <version>
```

**Example:**
```bash
# After releasing v0.3.1
./scripts/update-homebrew-formula.sh 0.3.1
```

**What it does:**
1. Fetches checksums from the GitHub release
2. Generates a new Homebrew formula with updated version and SHA256 hashes
3. Clones the homebrew-tap repository
4. Updates the formula file
5. Commits the changes with a descriptive message
6. Prompts for confirmation before pushing

**Prerequisites:**
- `gh` CLI installed and authenticated
- Write access to `scttfrdmn/homebrew-tap` repository
- The version must already be released on GitHub

**Notes:**
- The script handles both macOS (Intel and Apple Silicon) and Linux (x86_64 and ARM64) platforms
- Checksums are automatically fetched from the release's checksums.txt file
- The formula is validated before committing
