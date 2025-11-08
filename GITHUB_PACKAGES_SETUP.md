# GitHub Packages Setup Guide

If you're seeing a **403 Forbidden** error when pushing Docker images to GitHub Container Registry (ghcr.io), follow these steps:

## Steps to Fix Package Permissions

### 1. Enable Package Permissions in Repository Settings

1. Go to your repository on GitHub: `https://github.com/s3-bucketbird/bucketbird`
2. Click **Settings** → **Actions** → **General**
3. Scroll down to **Workflow permissions**
4. Select **"Read and write permissions"**
5. Check **"Allow GitHub Actions to create and approve pull requests"** (optional but recommended)
6. Click **Save**

### 2. Verify GitHub Container Registry is Public (Optional)

After the first successful push, you can make your packages public:

1. Go to your GitHub profile/organization
2. Click **Packages** tab
3. Find `bucketbird-backend` and `bucketbird-web` packages
4. Click on each package → **Package settings**
5. Scroll to **Danger Zone** → **Change visibility** → Make public

### 3. Expected Image Names

The GitHub Actions workflow will push images to:
- `ghcr.io/s3-bucketbird/bucketbird-backend:0.1.0`
- `ghcr.io/s3-bucketbird/bucketbird-backend:latest`
- `ghcr.io/s3-bucketbird/bucketbird-web:0.1.0`
- `ghcr.io/s3-bucketbird/bucketbird-web:latest`

These match your Kamal deployment configuration.

### 4. Test the Workflow

After updating the permissions:

1. Delete the failed tag (if it exists):
   ```bash
   git tag -d v0.1.0
   git push origin :refs/tags/v0.1.0
   ```

2. Create the tag again and push:
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```

3. Monitor the workflow: `https://github.com/s3-bucketbird/bucketbird/actions`

## Alternative: Use Personal Access Token (PAT)

If the above doesn't work, you can use a Personal Access Token instead:

1. Create a PAT with `write:packages` scope:
   - Go to **Settings** → **Developer settings** → **Personal access tokens** → **Tokens (classic)**
   - Click **Generate new token (classic)**
   - Name it "GitHub Packages"
   - Select scope: `write:packages`, `read:packages`, `delete:packages`
   - Generate and copy the token

2. Add it as a repository secret:
   - Go to repository **Settings** → **Secrets and variables** → **Actions**
   - Click **New repository secret**
   - Name: `GHCR_TOKEN`
   - Value: Paste your PAT
   - Click **Add secret**

3. Update `.github/workflows/release.yml`:
   ```yaml
   - name: Log in to GitHub Container Registry
     uses: docker/login-action@v3
     with:
       registry: ${{ env.REGISTRY }}
       username: ${{ github.actor }}
       password: ${{ secrets.GHCR_TOKEN }}  # Changed from GITHUB_TOKEN
   ```

## Troubleshooting

### Image names must be lowercase
- ✅ Correct: `ghcr.io/s3-bucketbird/bucketbird-backend`
- ❌ Wrong: `ghcr.io/S3-BucketBird/bucketbird-backend`

### Verify permissions
Check that the workflow has the required permissions in the job definition:
```yaml
permissions:
  contents: write
  packages: write
```

### Check workflow logs
Look for specific error messages in the GitHub Actions logs to diagnose issues.

## Current Configuration

Your current setup:
- **Organization**: `s3-bucketbird`
- **Registry**: `ghcr.io`
- **Backend Image**: `ghcr.io/s3-bucketbird/bucketbird-backend`
- **Frontend Image**: `ghcr.io/s3-bucketbird/bucketbird-web`
- **Kamal Config**: Correctly configured to pull from these images
