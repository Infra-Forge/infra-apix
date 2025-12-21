GO ?= go
GOCOVER ?= $(GO) test ./... -coverprofile=coverage.out -coverpkg=./...
MIN_COVERAGE ?= 85
SPEC_FILE ?= docs/openapi.yaml
SPEC_TITLE ?= Apix Library
SPEC_VERSION ?= 0.3.0
SPEC_SERVERS ?= https://api.example.com

.PHONY: all fmt lint test cover cover-html build clean help
.PHONY: generate-spec check-spec spec-guard install-hooks ci

all: fmt lint test

help:
	@echo "Apix Library - Available Make Targets"
	@echo ""
	@echo "Development:"
	@echo "  make fmt           - Format Go code"
	@echo "  make lint          - Run linters"
	@echo "  make test          - Run tests"
	@echo "  make cover         - Run tests with coverage (min $(MIN_COVERAGE)%)"
	@echo "  make cover-html    - Generate HTML coverage report"
	@echo "  make build         - Build library and CLI"
	@echo ""
	@echo "OpenAPI Spec:"
	@echo "  make generate-spec - Generate OpenAPI spec to $(SPEC_FILE)"
	@echo "  make check-spec    - Check for spec drift (CI-friendly)"
	@echo "  make spec-guard    - Alias for check-spec"
	@echo ""
	@echo "Git Hooks:"
	@echo "  make install-hooks - Install pre-commit hook for spec drift detection"
	@echo ""
	@echo "CI/CD:"
	@echo "  make ci            - Run all CI checks (fmt, lint, test, check-spec)"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean         - Remove generated files"
	@echo ""
	@echo "Configuration (override with make VAR=value):"
	@echo "  SPEC_FILE=$(SPEC_FILE)"
	@echo "  SPEC_TITLE=$(SPEC_TITLE)"
	@echo "  SPEC_VERSION=$(SPEC_VERSION)"
	@echo "  MIN_COVERAGE=$(MIN_COVERAGE)"

fmt:
	$(GO) fmt ./...

lint:
	@echo "(lint placeholder)"

test:
	$(GO) test ./...

cover:
	$(GOCOVER)
	$(GO) tool cover -func=coverage.out > coverage.txt
	@cov=$$(tail -n1 coverage.txt | awk '{print $$3}' | sed 's/%//'); \
	if [ $$cov -lt $(MIN_COVERAGE) ]; then \
		echo "Coverage below $(MIN_COVERAGE)%: $$cov"; \
		rm -f coverage.out coverage.txt; \
		exit 1; \
	fi
	@echo "Coverage: $$cov%"

cover-html: cover
	$(GO) tool cover -html=coverage.out -o coverage.html

build:
	$(GO) build ./...

clean:
	rm -f coverage.out coverage.html coverage.txt

# OpenAPI Spec Generation
generate-spec:
	@echo "📝 Generating OpenAPI spec..."
	@mkdir -p $(dir $(SPEC_FILE))
	@$(GO) run ./cmd/apix generate \
		--title "$(SPEC_TITLE)" \
		--version "$(SPEC_VERSION)" \
		--servers "$(SPEC_SERVERS)" \
		--out $(SPEC_FILE)
	@echo "✅ Generated: $(SPEC_FILE)"

# Check for spec drift (CI-friendly)
check-spec:
	@echo "🔍 Checking OpenAPI spec drift..."
	@$(GO) run ./cmd/apix spec-guard --existing $(SPEC_FILE)
	@echo "✅ OpenAPI spec is up to date"

# Alias for backwards compatibility
spec-guard: check-spec

# Install git hooks
install-hooks:
	@echo "🔧 Installing git hooks..."
	@mkdir -p .git/hooks
	@cp .githooks/pre-commit .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "✅ Pre-commit hook installed"
	@echo ""
	@echo "The hook will check for OpenAPI spec drift before each commit."
	@echo "To bypass: git commit --no-verify (NOT RECOMMENDED)"

# CI target - runs all checks
ci: fmt lint test check-spec
	@echo "✅ All CI checks passed!"

