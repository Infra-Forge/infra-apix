GO ?= go
GOCOVER ?= $(GO) test ./... -coverprofile=coverage.out -coverpkg=./...
MIN_COVERAGE ?= 85

.PHONY: all fmt lint test cover cover-html build clean spec-guard

all: fmt lint test

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
	rm -f coverage.out coverage.html

spec-guard:
	go run ./cmd/apix spec-guard

