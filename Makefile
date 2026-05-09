SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c

BIN          := kafko
PKG          := github.com/darioajr/kafko
LDFLAGS_PKG  := $(PKG)/internal/cli

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
  -X $(LDFLAGS_PKG).Version=$(VERSION) \
  -X $(LDFLAGS_PKG).Commit=$(COMMIT) \
  -X $(LDFLAGS_PKG).Date=$(DATE)

# Container engine: docker if available, otherwise podman. Override with
# `make docker CONTAINER_ENGINE=podman`.
CONTAINER_ENGINE ?= $(shell command -v docker >/dev/null 2>&1 && echo docker || (command -v podman >/dev/null 2>&1 && echo podman))

# When goreleaser runs the docker build step, it shells out to a binary named
# `docker`. SNAPSHOT_BIN_DIR is a temp PATH entry where we drop a `docker`
# symlink pointing at podman, so goreleaser can run unchanged.
SNAPSHOT_BIN_DIR := $(CURDIR)/.tmp/bin

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "}; /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: tidy
tidy: ## go mod tidy
	go mod tidy

.PHONY: build
build: ## Build the binary into ./bin
	@mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/$(BIN) ./cmd/kafko

.PHONY: install
install: ## Install kafko to $GOBIN
	CGO_ENABLED=0 go install -trimpath -ldflags="$(LDFLAGS)" ./cmd/kafko

.PHONY: test
test: ## Run unit tests with race detector
	go test -race -count=1 ./...

.PHONY: cover
cover: ## Run tests with coverage report
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

.PHONY: vet
vet: ## go vet
	go vet ./...

.PHONY: lint
lint: ## golangci-lint
	golangci-lint run

.PHONY: fmt
fmt: ## gofmt + goimports
	gofmt -w .

.PHONY: engine
engine: ## Show which container engine will be used
	@echo "CONTAINER_ENGINE=$(CONTAINER_ENGINE)"
	@if [ -z "$(CONTAINER_ENGINE)" ]; then \
	  echo "no container engine found (install docker or podman)"; exit 1; \
	fi

.PHONY: docker
docker: engine ## Build local container image (auto-detects docker or podman)
	$(CONTAINER_ENGINE) build \
	  --build-arg VERSION=$(VERSION) \
	  --build-arg COMMIT=$(COMMIT) \
	  --build-arg DATE=$(DATE) \
	  -t $(BIN):$(VERSION) -t $(BIN):latest .

.PHONY: snapshot
snapshot: ## goreleaser local snapshot (binaries only, skips container build)
	goreleaser release --snapshot --clean --skip=docker

.PHONY: snapshot-full
snapshot-full: engine ## goreleaser local snapshot including container images
ifeq ($(CONTAINER_ENGINE),podman)
	@mkdir -p $(SNAPSHOT_BIN_DIR)
	@ln -sf "$$(command -v podman)" $(SNAPSHOT_BIN_DIR)/docker
	@echo "using podman via shim at $(SNAPSHOT_BIN_DIR)/docker"
	PATH="$(SNAPSHOT_BIN_DIR):$$PATH" goreleaser release --snapshot --clean
else
	goreleaser release --snapshot --clean
endif

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf bin dist coverage.out .tmp
