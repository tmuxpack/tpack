VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
BINARY := tpm-go

.PHONY: build test test-unit test-integration test-e2e vet lint clean-build

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

lint: vet
	@if command -v staticcheck >/dev/null 2>&1; then staticcheck ./...; fi

coverage:
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -func=coverage.out

clean-build:
	rm -rf dist/ coverage.out
