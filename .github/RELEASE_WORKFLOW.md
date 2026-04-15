# GitHub Actions Release Workflow

This project uses automated release tagging via GitHub Actions with semantic versioning.

## Creating a Release

Releases are automatically created when merging PRs to the `main` branch with appropriate labels:

### Release Labels

- **`release:major`** - Breaking API changes (e.g., `v0.1.0` → `v1.0.0`)
- **`release:minor`** - New features, backward compatible (e.g., `v0.1.0` → `v0.2.0`)
- **`release:patch`** - Bug fixes, backward compatible (e.g., `v0.1.0` → `v0.1.1`)

### Workflow

1. Create a feature branch from `dev`
2. Make your changes
3. Create a pull request to `main`
4. Add the appropriate release label
5. Merge after CI passes
6. Tag is automatically created and pushed

### Example

```bash
# Create feature branch
git checkout -b feature/new-endpoint

# Make changes and commit
git commit -m "feat: add new message endpoint"

# Push and create PR
git push origin feature/new-endpoint

# On GitHub, add label "release:minor" and merge
# Tag v0.2.0 is automatically created
```

## Workflows

### Release Workflow (`release.yml`)
- Triggers on push to `main`
- Creates Git tags based on PR labels
- Handles edge cases (no PR, no labels, already tagged)

### CI Workflow (`ci.yml`)
- Runs on all pushes and PRs
- Tests against Go 1.23, 1.24, and 1.25
- Caches dependencies for speed

### Coverage Workflow (`coverage.yml`)
- Tracks test coverage
- Fails if coverage drops below 90%
- Uploads coverage reports for PRs

### Go Modules Workflow (`go-modules.yml`)
- Notifies Go module proxy of new releases
- Ensures `go get` works with new versions

## Safety Features

- ✅ Skips if no PR associated with commit
- ✅ Skips if no release label found
- ✅ Fails if multiple release labels
- ✅ Skips if commit already tagged
- ✅ Prevents duplicate tags

## Rollback

If incorrect tag is created:

```bash
# Delete tag locally and remotely
git tag -d vX.Y.Z
git push origin :refs/tags/vX.Y.Z

# Fix issues and create new PR with correct label
```

## Verification

After a release, verify:

```bash
# Check tag was created
git tag -l

# Verify Go module version
go list -m github.com/glennprays/whatsapp-gateway-sdk-go@vX.Y.Z

# Test importing
go get github.com/glennprays/whatsapp-gateway-sdk-go@vX.Y.Z
```
