# Release Process

This document describes the process for releasing new versions of BucketBird.

## Overview

BucketBird follows [Semantic Versioning](https://semver.org/):
- **MAJOR** version (X.0.0): Incompatible API changes
- **MINOR** version (0.X.0): New features, backwards compatible
- **PATCH** version (0.0.X): Bug fixes, backwards compatible

Current version: **0.1.0** (pre-1.0 means API may still change)

## Release Workflow

### 1. Prepare the Release

#### Update CHANGELOG.md

Edit `CHANGELOG.md` and move items from `[Unreleased]` to a new version section:

```markdown
## [Unreleased]

## [0.2.0] - 2025-11-15

### Added
- New feature X
- New feature Y

### Fixed
- Bug fix Z
```

Categories to use:
- **Added**: New features
- **Changed**: Changes to existing functionality
- **Deprecated**: Soon-to-be removed features
- **Removed**: Removed features
- **Fixed**: Bug fixes
- **Security**: Security improvements

#### Review Changes

```bash
# Check what's changed since last release
git log v0.1.0..HEAD --oneline

# Review uncommitted changes
git status
git diff
```

### 2. Bump Version Numbers

Run the version bump script with the new version:

```bash
./scripts/bump-version.sh 0.2.0
```

This script will:
- Update `VERSION` file
- Update `frontend/package.json`
- Update `CHANGELOG.md` (convert `[Unreleased]` to the new version)

**Important:** The version should NOT include the 'v' prefix (use `0.2.0`, not `v0.2.0`)

### 3. Commit and Tag

Review the changes made by the bump script:

```bash
git diff
```

If everything looks good, commit and tag:

```bash
# Commit version bump
git add -A
git commit -m "Bump version to 0.2.0"

# Create annotated tag (note the 'v' prefix here)
git tag -a v0.2.0 -m "Release v0.2.0"

# Verify the tag
git tag -l
git show v0.2.0
```

### 4. Push to Trigger Release

Push both the commit and the tag:

```bash
# Push the commit
git push origin main

# Push the tag (this triggers the release workflow)
git push origin v0.2.0

# Or push everything at once
git push && git push --tags
```

### 5. Monitor the Release

GitHub Actions will automatically:
1. Build multi-arch Docker images (AMD64 + ARM64)
2. Push images to GitHub Container Registry with version tag + `latest`
3. Create a GitHub Release with changelog notes

Monitor progress at: `https://github.com/s3-bucketbird/bucketbird/actions`

Expected images:
- `ghcr.io/s3-bucketbird/bucketbird-backend:0.2.0`
- `ghcr.io/s3-bucketbird/bucketbird-backend:latest`
- `ghcr.io/s3-bucketbird/bucketbird-web:0.2.0`
- `ghcr.io/s3-bucketbird/bucketbird-web:latest`

### 6. Deploy the Release

Once the release workflow completes successfully, deploy using Kamal:

#### Deploy Backend

```bash
# Deploy backend with latest version
kamal deploy -d backend

# Or deploy specific version
kamal deploy -d backend --version 0.2.0
```

#### Deploy Frontend

```bash
# Deploy frontend with latest version
kamal deploy -d web

# Or deploy specific version
kamal deploy -d web --version 0.2.0
```

#### Verify Deployment

```bash
# Check backend health
curl https://api.bucketbird.xyz/healthz

# Check frontend health
curl https://demo.bucketbird.xyz/healthz

# View backend logs
kamal app logs -d backend

# View frontend logs
kamal app logs -d web
```

### 7. Post-Release

- Verify the application works correctly in production
- Update any external documentation if needed
- Announce the release (social media, blog, etc.)
- Close any GitHub issues resolved in this release

## Hotfix Process

For urgent bug fixes:

1. Create a hotfix branch from the release tag:
   ```bash
   git checkout -b hotfix/0.1.1 v0.1.0
   ```

2. Make the fix and commit:
   ```bash
   git commit -am "Fix critical bug in XYZ"
   ```

3. Follow the normal release process starting from step 2:
   ```bash
   ./scripts/bump-version.sh 0.1.1
   git add -A
   git commit -m "Bump version to 0.1.1"
   git tag -a v0.1.1 -m "Release v0.1.1"
   ```

4. Merge back to main:
   ```bash
   git checkout main
   git merge hotfix/0.1.1
   git push && git push --tags
   ```

## Pre-releases

For beta/RC versions, use the pre-release suffix:

```bash
# Example: 1.0.0-beta.1
./scripts/bump-version.sh 1.0.0-beta.1
```

GitHub will automatically mark these as pre-releases.

## Rollback

If something goes wrong after deployment:

```bash
# Rollback to previous version using Kamal
kamal rollback -d backend
kamal rollback -d web

# Or redeploy a specific older version
kamal deploy -d backend --version 0.1.0
kamal deploy -d web --version 0.1.0
```

## Troubleshooting

### GitHub Actions fails to build

- Check the workflow logs in GitHub Actions tab
- Verify Docker build works locally:
  ```bash
  docker build -t test-backend ./backend
  docker build -t test-frontend ./frontend
  ```

### Tag already exists

```bash
# Delete local tag
git tag -d v0.2.0

# Delete remote tag
git push origin :refs/tags/v0.2.0

# Create tag again
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

### Failed to push to registry

- Verify GITHUB_TOKEN permissions in repository settings
- Check that GitHub Packages is enabled
- Ensure you're not hitting rate limits

## Version History

- **v0.1.0** (2025-11-08): Initial MVP release

## References

- [Semantic Versioning](https://semver.org/)
- [Keep a Changelog](https://keepachangelog.com/)
- [Kamal Documentation](https://kamal-deploy.org/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
