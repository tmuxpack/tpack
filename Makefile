VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
BINARY := tpm-go

.PHONY: build test test-unit test-integration test-e2e vet lint lint-fix coverage clean-build setup-hooks

build:
	go build $(LDFLAGS) -o dist/$(BINARY) ./cmd/tpm

test: test-unit

test-unit:
	go test -race -count=1 ./internal/...

test-short:
	go test -short -race -count=1 ./...

test-integration:
	go test -race -count=1 ./tests/integration/...

test-e2e:
	go test -race -count=1 ./tests/e2e/...

test-all: test-unit test-integration test-e2e

vet:
	go vet ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	}
	golangci-lint run

lint-fix:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	}
	golangci-lint run --fix

coverage:
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -func=coverage.out

clean-build:
	rm -rf dist/ coverage.out

setup-hooks:
	git config core.hooksPath .githooks
