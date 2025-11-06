# Using apix as a Private Go Module

This guide explains how to use `apix` as a private Go module within your organization before making it public.

## Repository Information

- **Module Path**: `github.com/Infra-Forge/apix`
- **Repository**: `git@github.com:Infra-Forge/infra-apix.git`
- **Status**: Private (organization-only)

## For Library Maintainers (You)

### 1. Tag Releases

Create semantic version tags for releases:

```bash
# Tag a release
git tag -a v0.1.0 -m "Initial private release"
git push origin v0.1.0

# Or for production-ready
git tag -a v1.0.0 -m "Production release: nested schemas, path normalization"
git push origin v1.0.0
```

### 2. Verify Module Path

Your `go.mod` is already correct:
```go
module github.com/Infra-Forge/apix
```

This matches your GitHub repository path. ✅

## For Users in Your Organization

### Method 1: Using GOPRIVATE (Recommended)

This is the cleanest approach for private modules.

#### Step 1: Configure GOPRIVATE

Tell Go that this module is private:

```bash
# For this specific repo
go env -w GOPRIVATE=github.com/Infra-Forge/apix

# Or for all Infra-Forge repos
go env -w GOPRIVATE=github.com/Infra-Forge/*

# Or for all your private repos
go env -w GOPRIVATE=github.com/Infra-Forge/*,github.com/YourOtherOrg/*
```

Verify:
```bash
go env GOPRIVATE
# Should output: github.com/Infra-Forge/*
```

#### Step 2: Configure Git Authentication

**Option A: SSH (Recommended)**

Ensure you have SSH keys set up with GitHub:

```bash
# Test SSH access
ssh -T git@github.com
# Should output: Hi username! You've successfully authenticated...

# Configure Git to use SSH for this repo
git config --global url."git@github.com:".insteadOf "https://github.com/"
```

**Option B: Personal Access Token (PAT)**

1. Create a GitHub Personal Access Token:
   - Go to GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
   - Click "Generate new token (classic)"
   - Select scopes: `repo` (full control of private repositories)
   - Copy the token (you won't see it again!)

2. Configure Git credentials:

```bash
# Option B1: Using .netrc (macOS/Linux)
cat >> ~/.netrc << EOF
machine github.com
login YOUR_GITHUB_USERNAME
password YOUR_PERSONAL_ACCESS_TOKEN
EOF

chmod 600 ~/.netrc

# Option B2: Using Git credential helper
git config --global credential.helper store
echo "https://YOUR_GITHUB_USERNAME:YOUR_PERSONAL_ACCESS_TOKEN@github.com" >> ~/.git-credentials
chmod 600 ~/.git-credentials
```

#### Step 3: Install the Module

In your project (e.g., `infranotes-module`):

```bash
cd ~/Documents/InfraForge/infranotes-module

# Install latest version
go get github.com/Infra-Forge/apix

# Or install specific version
go get github.com/Infra-Forge/apix@v1.0.0

# Update go.mod
go mod tidy
```

Your `go.mod` will now have:
```go
require (
    github.com/Infra-Forge/apix v1.0.0
    // ... other dependencies
)
```

### Method 2: Using Replace Directive (Development)

For local development or testing unreleased changes:

```bash
cd ~/Documents/InfraForge/infranotes-module
```

Edit `go.mod`:
```go
module github.com/StackCatalyst/infranotes-module

go 1.24.2

require (
    github.com/Infra-Forge/apix v1.0.0
)

// Use local version for development
replace github.com/Infra-Forge/apix => ../infra-apix
```

Then:
```bash
go mod tidy
```

**Important**: Remove the `replace` directive before committing to production!

## Verification

Test that it works:

```bash
cd ~/Documents/InfraForge/infranotes-module

# This should succeed without prompts
go get github.com/Infra-Forge/apix@v1.0.0

# Verify it's in go.mod
grep "Infra-Forge/apix" go.mod
```

## Team Setup Instructions

Share these instructions with your team:

### Quick Setup for Team Members

```bash
# 1. Configure GOPRIVATE
go env -w GOPRIVATE=github.com/Infra-Forge/*

# 2. Ensure SSH access to GitHub
ssh -T git@github.com

# 3. Configure Git to use SSH
git config --global url."git@github.com:".insteadOf "https://github.com/"

# 4. In your project, install the module
go get github.com/Infra-Forge/apix@v1.0.0
```

## CI/CD Setup

For GitHub Actions, GitLab CI, or other CI systems:

### GitHub Actions

```yaml
# .github/workflows/build.yml
name: Build

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Configure Git for private modules
        run: |
          git config --global url."https://${{ secrets.GH_PAT }}@github.com/".insteadOf "https://github.com/"
      
      - name: Set GOPRIVATE
        run: go env -w GOPRIVATE=github.com/Infra-Forge/*
      
      - name: Build
        run: go build ./...
```

Add `GH_PAT` secret to your repository with a Personal Access Token.

### Docker Build

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

# Install git
RUN apk add --no-cache git

WORKDIR /app

# Configure Git for private modules
ARG GITHUB_TOKEN
RUN git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"

# Set GOPRIVATE
ENV GOPRIVATE=github.com/Infra-Forge/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o app ./cmd/...

FROM alpine:latest
COPY --from=builder /app/app /app
CMD ["/app"]
```

Build with:
```bash
docker build --build-arg GITHUB_TOKEN=your_token_here -t myapp .
```

## Troubleshooting

### Error: "terminal prompts disabled"

**Cause**: Go can't authenticate to GitHub.

**Solution**: Configure Git authentication (see Step 2 above).

### Error: "410 Gone" or "404 Not Found"

**Cause**: GOPRIVATE not set, Go is trying to use proxy.

**Solution**:
```bash
go env -w GOPRIVATE=github.com/Infra-Forge/*
```

### Error: "Permission denied (publickey)"

**Cause**: SSH keys not configured.

**Solution**:
```bash
# Generate SSH key if you don't have one
ssh-keygen -t ed25519 -C "your_email@example.com"

# Add to GitHub: Settings → SSH and GPG keys → New SSH key
cat ~/.ssh/id_ed25519.pub
```

### Module not updating

**Solution**:
```bash
# Clear module cache
go clean -modcache

# Re-download
go get github.com/Infra-Forge/apix@v1.0.0
```

## Making the Module Public (Future)

When you're ready to make it public:

1. Change repository visibility on GitHub to Public
2. Remove GOPRIVATE configuration (optional)
3. Announce the release
4. Users can install without special configuration:
   ```bash
   go get github.com/Infra-Forge/apix
   ```

## Summary

**For you (maintainer):**
- Tag releases: `git tag v1.0.0 && git push origin v1.0.0`

**For your team:**
```bash
go env -w GOPRIVATE=github.com/Infra-Forge/*
git config --global url."git@github.com:".insteadOf "https://github.com/"
go get github.com/Infra-Forge/apix@v1.0.0
```

That's it! Your private module is now installable across your organization.

