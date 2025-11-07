# apix Library Distribution Strategy

## Overview
This document outlines how to make the `apix` library installable and usable in Go projects, including GitHub Actions CI/CD pipelines for automated testing, linting, and releases.

## 1. Go Module Publishing (Automatic via GitHub)

### How Go Module Discovery Works
1. **Git Tags**: Create semantic version tags (e.g., `v0.2.0`)
2. **GitHub Releases**: Push tags to GitHub
3. **Go Proxy**: `proxy.golang.org` automatically indexes your module
4. **pkg.go.dev**: Documentation automatically appears at `pkg.go.dev/github.com/Infra-Forge/apix`

### Publishing Steps
```bash
# 1. Ensure code is clean
go mod tidy
go test ./...

# 2. Create a semantic version tag
git tag v0.2.0

# 3. Push tag to GitHub
git push origin v0.2.0

# 4. Go proxy indexes automatically (takes ~5 minutes)
# Users can now run: go get github.com/Infra-Forge/apix@v0.2.0
```

### Semantic Versioning for apix
- **v0.2.0** (Current): Milestone 2 complete, 5 framework adapters
- **v0.3.0** (Next): Typed query/header params, middleware detection, examples
- **v1.0.0** (Future): Stable API, production-ready

## 2. GitHub Actions CI/CD Pipelines

### Pipeline 1: Test & Lint (on every push/PR)
**File**: `.github/workflows/test.yml`

Runs:
- Go tests across multiple versions (1.24, 1.25)
- Code coverage analysis
- Linting with golangci-lint
- Coverage badge generation

### Pipeline 2: Release (on version tag)
**File**: `.github/workflows/release.yml`

Runs when tag is pushed (e.g., `git push origin v0.2.0`):
- Validates tests pass
- Builds CLI binaries (Linux, macOS, Windows)
- Creates GitHub Release with binaries
- Publishes to pkg.go.dev

### Pipeline 3: Documentation (on release)
**File**: `.github/workflows/docs.yml`

Runs after release:
- Generates API documentation
- Updates README badges
- Publishes to GitHub Pages (optional)

## 3. CLI Tool Distribution (apix command)

### Current Status
- CLI tool at `cmd/apix/main.go`
- Commands: `apix generate`, `apix spec-guard`

### Distribution Methods

#### Method 1: go install (Recommended for users)
```bash
go install github.com/Infra-Forge/apix/cmd/apix@latest
go install github.com/Infra-Forge/apix/cmd/apix@v0.2.0
```

#### Method 2: GoReleaser (for binary distribution)
- Builds cross-platform binaries
- Creates installers for macOS (Homebrew), Linux, Windows
- Publishes to GitHub Releases
- Enables: `brew install apix` (via Homebrew tap)

#### Method 3: Docker Image
- Publish Docker image: `ghcr.io/infra-forge/apix:v0.2.0`
- Users: `docker run ghcr.io/infra-forge/apix:v0.2.0 generate`

## 4. Recommended Implementation Order

### Phase 1: Basic Release Pipeline (Week 1)
- [ ] Create `.github/workflows/test.yml` (test + lint)
- [ ] Create `.github/workflows/release.yml` (GitHub Releases)
- [ ] Add coverage badge to README
- [ ] Tag v0.2.0 and test pipeline

### Phase 2: Enhanced Distribution (Week 2)
- [ ] Add GoReleaser config (`.goreleaser.yaml`)
- [ ] Build cross-platform binaries
- [ ] Create Homebrew tap (optional)
- [ ] Document installation methods in README

### Phase 3: Documentation & Examples (Week 3)
- [ ] Create `.github/workflows/docs.yml`
- [ ] Add example applications to release notes
- [ ] Create migration guide from swaggo
- [ ] Publish to GitHub Pages

## 5. Key Files to Create

```
.github/workflows/
├── test.yml              # Test, lint, coverage
├── release.yml           # Create GitHub Release
└── docs.yml              # Documentation generation

.goreleaser.yaml          # Binary distribution config
CHANGELOG.md              # Release notes template
```

## 6. Benefits of This Approach

✅ **Zero Manual Steps**: Automated from git tag to pkg.go.dev
✅ **Multi-Platform**: Users get binaries for Linux, macOS, Windows
✅ **Version Control**: Semantic versioning with git tags
✅ **Quality Gates**: Tests + linting before release
✅ **Documentation**: Auto-indexed on pkg.go.dev
✅ **User Friendly**: `go get` and `go install` work seamlessly

## 7. User Experience After Setup

### For Library Users
```bash
# Get latest version
go get github.com/Infra-Forge/apix

# Get specific version
go get github.com/Infra-Forge/apix@v0.2.0

# Use in code
import "github.com/Infra-Forge/apix"
```

### For CLI Users
```bash
# Install latest CLI
go install github.com/Infra-Forge/apix/cmd/apix@latest

# Use CLI
apix generate --title "My API" --out docs/openapi.yaml
```

## 8. Next Steps

1. Review this strategy with team
2. Create GitHub Actions workflows
3. Set up GoReleaser (optional but recommended)
4. Tag v0.2.0 and test the pipeline
5. Document in README and CONTRIBUTING.md

