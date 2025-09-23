# Pull Request

## Description

<!-- Provide a brief description of the changes in this PR -->

## Type of Change

Please check the type of change your PR introduces:

- [ ] 🐛 **Bug fix** (non-breaking change which fixes an issue)
- [ ] ✨ **New feature** (non-breaking change which adds functionality)
- [ ] 💥 **Breaking change** (fix or feature that would cause existing functionality to not work as expected)
- [ ] 📚 **Documentation** (updates to documentation only)
- [ ] 🔧 **Refactoring** (code change that neither fixes a bug nor adds a feature)
- [ ] ⚡ **Performance** (code change that improves performance)
- [ ] 🧪 **Tests** (adding missing tests or correcting existing tests)
- [ ] 🏗️ **CI/Build** (changes to CI configuration or build scripts)

## Related Issues

<!-- Link any related issues using keywords like "Fixes #123" or "Relates to #456" -->

- Fixes #
- Relates to #

## Changes Made

<!-- Describe the specific changes made in this PR -->

### Added
-

### Changed
-

### Removed
-

## Testing

<!-- Describe how you tested your changes -->

### Test Coverage
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing performed

### Test Commands
```bash
# Commands used to test the changes
go test ./...
go test ./... -cover
```

### Test Results
<!-- Paste relevant test output or describe test results -->

```
# Paste test output here
```

## Code Quality

### Pre-submission Checklist
- [ ] Code follows the project's coding standards
- [ ] Self-review of code completed
- [ ] Comments added for complex logic
- [ ] No unnecessary console logs or debug code
- [ ] Error handling implemented where appropriate

### Quality Checks
- [ ] `go fmt` - Code is properly formatted
- [ ] `go vet` - No static analysis issues
- [ ] `gocyclo -over 15` - No overly complex functions
- [ ] Tests pass - `go test ./...`
- [ ] Linting passes (if pre-commit hooks installed)

## Documentation

- [ ] Updated relevant documentation
- [ ] Updated CHANGELOG.md (if applicable)
- [ ] Added/updated code comments
- [ ] Updated CLI help text (if applicable)

## Backwards Compatibility

<!-- For breaking changes, describe the impact and migration path -->

- [ ] This change is backwards compatible
- [ ] This change includes breaking changes (explain below)

### Breaking Changes (if applicable)
<!-- Describe what breaks and how users should migrate -->

## Screenshots/Examples

<!-- If applicable, add screenshots or command examples showing the changes -->

```bash
# Example usage of new feature
aws-jupyter command --new-flag value
```

## Reviewer Notes

<!-- Any specific areas you'd like reviewers to focus on -->

### Areas of Focus
- [ ] Algorithm/logic correctness
- [ ] Error handling
- [ ] Performance implications
- [ ] Security considerations
- [ ] User experience

### Questions for Reviewers
<!-- Any specific questions you have for reviewers -->

## Deployment Notes

<!-- Any special considerations for deployment -->

- [ ] No special deployment considerations
- [ ] Requires environment variable changes
- [ ] Requires configuration updates
- [ ] Requires database migrations (if applicable)

---

## Checklist

### Before Requesting Review
- [ ] I have performed a self-review of my code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes

### After Review
- [ ] I have addressed all review comments
- [ ] I have resolved any merge conflicts
- [ ] All CI checks are passing

---

**Additional Notes:**
<!-- Any other information that would be helpful for reviewers -->

/cc @scttfrdmn <!-- Tag maintainers if needed -->