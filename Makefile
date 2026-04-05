# =============================================================================
# MTPanel Makefile
# =============================================================================
# Usage:
#   make                    — build local binary
#   make build              — build for current OS/arch
#   make build-all          — cross-compile for linux/amd64 and linux/arm64
#   make build-frontend     — build SvelteKit frontend
#   make embed-frontend     — build frontend + embed into Go binary
#   make release VERSION=v1.2.3  — tag + push release
#   make docker             — build Docker image
#   make clean              — remove build artefacts
#   make lint               — run golangci-lint
#   make test               — run Go tests
# =============================================================================

# Project metadata
MODULE    := github.com/mtpanel/mtpanel
BINARY    := mtpanel
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Directories
BUILD_DIR  := dist
WEB_DIR    := web
EMBED_DIR  := internal/web   # where embedded FS lives in Go code

# Go build flags
LDFLAGS := -s -w \
  -X '$(MODULE)/internal/version.Version=$(VERSION)' \
  -X '$(MODULE)/internal/version.Commit=$(COMMIT)'   \
  -X '$(MODULE)/internal/version.BuildDate=$(BUILD_DATE)'
GO_FLAGS  := -trimpath -ldflags "$(LDFLAGS)"

# Docker
DOCKER_IMAGE  := ghcr.io/mtpanel/mtpanel
DOCKER_TAG    ?= $(VERSION)

# Cross-compile targets
PLATFORMS := linux/amd64 linux/arm64

# Colours (GNU make)
RESET  := \033[0m
BOLD   := \033[1m
GREEN  := \033[32m
CYAN   := \033[36m
YELLOW := \033[33m

.PHONY: all build build-all build-frontend embed-frontend \
        release docker clean lint test check fmt vet \
        install-tools help

# ---------------------------------------------------------------------------
# Default target
# ---------------------------------------------------------------------------
all: embed-frontend build

# ---------------------------------------------------------------------------
# Build — local OS/arch
# ---------------------------------------------------------------------------
build:
	@printf "$(CYAN)$(BOLD)Building $(BINARY) $(VERSION)$(RESET)\n"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(GO_FLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/$(BINARY)
	@printf "$(GREEN)  => $(BUILD_DIR)/$(BINARY)$(RESET)\n"

# ---------------------------------------------------------------------------
# Cross-compile for all release platforms
# ---------------------------------------------------------------------------
build-all: embed-frontend
	@printf "$(CYAN)$(BOLD)Cross-compiling for all platforms$(RESET)\n"
	@mkdir -p $(BUILD_DIR)
	@$(foreach PLATFORM,$(PLATFORMS), \
		$(eval OS   := $(word 1,$(subst /, ,$(PLATFORM)))) \
		$(eval ARCH := $(word 2,$(subst /, ,$(PLATFORM)))) \
		printf "  $(CYAN)Building $(OS)/$(ARCH)...$(RESET)\n"; \
		CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) \
		  go build $(GO_FLAGS) \
		  -o $(BUILD_DIR)/$(BINARY)-$(OS)-$(ARCH) \
		  ./cmd/$(BINARY) || exit 1; \
		sha256sum $(BUILD_DIR)/$(BINARY)-$(OS)-$(ARCH) \
		  > $(BUILD_DIR)/$(BINARY)-$(OS)-$(ARCH).sha256; \
		printf "  $(GREEN)=> $(BUILD_DIR)/$(BINARY)-$(OS)-$(ARCH)$(RESET)\n"; \
	)
	@printf "$(GREEN)$(BOLD)All binaries built$(RESET)\n"

# ---------------------------------------------------------------------------
# Frontend — SvelteKit
# ---------------------------------------------------------------------------
build-frontend:
	@printf "$(CYAN)$(BOLD)Building SvelteKit frontend$(RESET)\n"
	@command -v node >/dev/null 2>&1 || { printf "$(YELLOW)node not found — skipping frontend build$(RESET)\n"; exit 0; }
	@cd $(WEB_DIR) && \
	  ( [ -d node_modules ] || npm ci --silent ) && \
	  npm run build
	@printf "$(GREEN)  Frontend built in $(WEB_DIR)/build$(RESET)\n"

# ---------------------------------------------------------------------------
# Embed frontend into Go binary
# ---------------------------------------------------------------------------
embed-frontend: build-frontend
	@printf "$(CYAN)$(BOLD)Embedding frontend into Go$(RESET)\n"
	@mkdir -p $(EMBED_DIR)
	@# Copy SvelteKit static build output into the embedded FS directory.
	@# The Go source file using //go:embed references this path.
	@if [ -d "$(WEB_DIR)/build" ]; then \
	  rm -rf $(EMBED_DIR)/static && \
	  cp -r $(WEB_DIR)/build $(EMBED_DIR)/static && \
	  printf "$(GREEN)  Embedded: $(EMBED_DIR)/static$(RESET)\n"; \
	else \
	  printf "$(YELLOW)  $(WEB_DIR)/build not found — binary will serve no frontend$(RESET)\n"; \
	fi

# ---------------------------------------------------------------------------
# Release — tag, build all, generate checksums
# ---------------------------------------------------------------------------
release: check
ifndef VERSION
	$(error VERSION is not set. Usage: make release VERSION=v1.2.3)
endif
	@printf "$(CYAN)$(BOLD)Releasing $(VERSION)$(RESET)\n"
	@git diff --quiet || { printf "$(YELLOW)Working tree is dirty — commit or stash changes first$(RESET)\n"; exit 1; }
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@git push origin $(VERSION)
	@$(MAKE) build-all VERSION=$(VERSION)
	@printf "$(GREEN)$(BOLD)Release $(VERSION) ready in $(BUILD_DIR)/$(RESET)\n"
	@printf "$(CYAN)  Upload binaries to GitHub Releases manually or via CI$(RESET)\n"

# ---------------------------------------------------------------------------
# Docker
# ---------------------------------------------------------------------------
docker: embed-frontend
	@printf "$(CYAN)$(BOLD)Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)$(RESET)\n"
	docker build \
	  --build-arg VERSION=$(VERSION) \
	  --build-arg COMMIT=$(COMMIT) \
	  --build-arg BUILD_DATE=$(BUILD_DATE) \
	  -t $(DOCKER_IMAGE):$(DOCKER_TAG) \
	  -t $(DOCKER_IMAGE):latest \
	  .
	@printf "$(GREEN)  Image: $(DOCKER_IMAGE):$(DOCKER_TAG)$(RESET)\n"

docker-push: docker
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):latest

# ---------------------------------------------------------------------------
# Quality
# ---------------------------------------------------------------------------
test:
	@printf "$(CYAN)$(BOLD)Running tests$(RESET)\n"
	go test -race -count=1 ./...

lint:
	@printf "$(CYAN)$(BOLD)Running golangci-lint$(RESET)\n"
	@command -v golangci-lint >/dev/null 2>&1 || { \
	  printf "$(YELLOW)golangci-lint not found — run: make install-tools$(RESET)\n"; exit 1; }
	golangci-lint run ./...

fmt:
	@printf "$(CYAN)$(BOLD)Formatting$(RESET)\n"
	gofmt -w -s .

vet:
	@printf "$(CYAN)$(BOLD)Running go vet$(RESET)\n"
	go vet ./...

check: fmt vet lint test

# ---------------------------------------------------------------------------
# Tools
# ---------------------------------------------------------------------------
install-tools:
	@printf "$(CYAN)$(BOLD)Installing development tools$(RESET)\n"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@printf "$(GREEN)  Tools installed to $(GOPATH)/bin$(RESET)\n"

# ---------------------------------------------------------------------------
# Clean
# ---------------------------------------------------------------------------
clean:
	@printf "$(CYAN)$(BOLD)Cleaning$(RESET)\n"
	rm -rf $(BUILD_DIR)
	rm -rf $(EMBED_DIR)/static
	@printf "$(GREEN)  Clean$(RESET)\n"

# ---------------------------------------------------------------------------
# Help
# ---------------------------------------------------------------------------
help:
	@printf "$(BOLD)MTPanel — available make targets$(RESET)\n\n"
	@printf "  $(CYAN)all$(RESET)              Build frontend + binary (default)\n"
	@printf "  $(CYAN)build$(RESET)            Build binary for current platform\n"
	@printf "  $(CYAN)build-all$(RESET)        Cross-compile linux/amd64 + linux/arm64\n"
	@printf "  $(CYAN)build-frontend$(RESET)   Build SvelteKit frontend\n"
	@printf "  $(CYAN)embed-frontend$(RESET)   Build and embed frontend into Go\n"
	@printf "  $(CYAN)release VERSION=vX.Y.Z$(RESET)  Tag and build release\n"
	@printf "  $(CYAN)docker$(RESET)           Build Docker image\n"
	@printf "  $(CYAN)docker-push$(RESET)      Push Docker image to registry\n"
	@printf "  $(CYAN)test$(RESET)             Run Go tests\n"
	@printf "  $(CYAN)lint$(RESET)             Run golangci-lint\n"
	@printf "  $(CYAN)fmt$(RESET)              Run gofmt\n"
	@printf "  $(CYAN)vet$(RESET)              Run go vet\n"
	@printf "  $(CYAN)check$(RESET)            fmt + vet + lint + test\n"
	@printf "  $(CYAN)install-tools$(RESET)    Install dev tools (golangci-lint, goimports)\n"
	@printf "  $(CYAN)clean$(RESET)            Remove build artefacts\n"
