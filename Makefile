# auth-accelerator — common developer tasks.
BINARY      := auth-accelerator
PKG         := github.com/cmalafis10/auth-accelerator
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS     := -s -w -X $(PKG)/cmd.version=$(VERSION)

.PHONY: build test vet check install snapshot golden clean

build: ## Build the binary with the version stamped in
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

test: ## Run the unit tests
	go test ./...

vet: ## go vet
	go vet ./...

check: vet test ## vet + test

install: ## Install to GOBIN
	go install -ldflags "$(LDFLAGS)" .

golden: ## Regenerate the render golden snapshots
	go test ./internal/render -run TestGolden -update

snapshot: ## Build a local release (binaries + image) without publishing
	goreleaser release --snapshot --clean

clean:
	rm -rf dist/ $(BINARY)
