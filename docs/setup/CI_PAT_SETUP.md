# CI Personal Access Token Setup

This document explains how to set up the `CR_PAT` secret required for Docker image pushes to GitHub Container Registry.

## Problem

The CI workflow fails with a 403 Forbidden error when pushing Docker images to `ghcr.io` because the default `GITHUB_TOKEN` lacks `write:packages` permission.

## Solution

Use a Personal Access Token (PAT) with the required permissions.

## Setup Instructions

### 1. Create a Personal Access Token

1. Go to GitHub Settings: https://github.com/settings/tokens
2. Click "Generate new token (classic)"
3. Configure the token:
   - **Token name**: `Nephoran CI Container Registry`
   - **Expiration**: Choose appropriate expiration (e.g., 90 days, 1 year, or no expiration)
   - **Scopes**: Select the following permissions:
     - ✅ `write:packages` - Upload packages to GitHub Package Registry
     - ✅ `read:packages` - Download packages from GitHub Package Registry
     - ✅ `delete:packages` - Delete packages from GitHub Package Registry (optional)

4. Click "Generate token"
5. **Copy the token immediately** - you won't be able to see it again

### 2. Add Secret to Repository

1. Go to your repository: https://github.com/thc1006/nephoran-intent-operator
2. Navigate to **Settings** → **Secrets and variables** → **Actions**
3. Click **"New repository secret"**
4. Configure the secret:
   - **Name**: `CR_PAT`
   - **Value**: Paste the PAT you generated in step 1
5. Click **"Add secret"**

### 3. Verify Setup

After adding the secret, the CI workflow will automatically use it for Docker registry authentication. You can verify by:

1. Pushing a commit to trigger the CI workflow
2. Check that the "Log in to Container Registry" step succeeds
3. Verify that the "Build and push container image" step completes without 403 errors

## Security Notes

- **Rotate regularly**: Set token expiration and rotate before it expires
- **Minimal permissions**: Only grant the permissions needed (`write:packages`, `read:packages`)
- **Monitor usage**: Review token usage in GitHub Settings → Personal access tokens
- **Revoke if compromised**: If the token is compromised, revoke it immediately and create a new one

## Troubleshooting

### Token Permission Issues
If you still get 403 errors:
- Verify the token has `write:packages` scope
- Check that the token hasn't expired
- Ensure you're using the correct secret name (`CR_PAT`)

### Token Expiration
- GitHub will send email notifications before token expiration
- Set up calendar reminders to rotate tokens proactively
- Consider using GitHub App tokens for longer-term solutions

### Alternative: GitHub App
For production environments, consider using a GitHub App instead of PAT for better security and longer token life.

## References

- [GitHub Packages Authentication](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry#authenticating-to-the-container-registry)
- [Creating Personal Access Tokens](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)
- [Using Secrets in GitHub Actions](https://docs.github.com/en/actions/security-guides/using-secrets-in-github-actions)