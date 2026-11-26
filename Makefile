# Single source of truth for tool versions; CI and the devcontainer read them from here.
GOLANGCI_LINT_VERSION := v2.11.4
GOIMPORTS_VERSION := v0.53.0
GOVULNCHECK_VERSION := v1.10.0
GORELEASER_VERSION := v2.20.3
# syft generates release SBOMs; the digest is for syft_<version>_linux_amd64.tar.gz from the upstream checksums file.
SYFT_VERSION := 1.57.2
SYFT_LINUX_AMD64_SHA256 := ad92bd3ec1b94a3df01d4d0aec199afa9c42b72e86da436ff5f837914d32706b
SYFT_BIN := $(CURDIR)/bin/syft-$(SYFT_VERSION)
LOCAL_PREFIX := github.com/solarwinds/swo-cli
# Keep in sync with the build targets in .goreleaser.yaml.
RELEASE_TARGETS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64

GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GOIMPORTS := go run golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)
# Installed for the host; go run would cross-compile it when GOOS is set.
GOVULNCHECK_BIN := $(CURDIR)/bin/govulncheck-$(GOVULNCHECK_VERSION)
GOVULNCHECK := $(GOVULNCHECK_BIN)/govulncheck

.PHONY: help
help: ## Show this help message
	@echo "Available targets:"
	@echo "  build          Build the swo binary"
	@echo "  test           Run all tests"
	@echo "  lint           Run linter and show issues"
	@echo "  fix-lint       Auto-fix formatting and some linting issues"
	@echo "  vuln           Scan release targets for known vulnerabilities (govulncheck)"
	@echo "  install-syft   Install the pinned syft for SBOM generation (Linux amd64 only)"
	@echo "  help           Show this help message"

# Build targets
.PHONY: build
build:
	go build -o swo ./cmd/swo
	chmod +x swo

.PHONY: test
test:
	go test -race ./...

.PHONY: print-golangci-lint-version
print-golangci-lint-version:
	@echo $(GOLANGCI_LINT_VERSION)

.PHONY: print-goreleaser-version
print-goreleaser-version:
	@echo $(GORELEASER_VERSION)

# Linux amd64 only (CI runners); prints the directory to add to PATH.
.PHONY: install-syft
install-syft:
	@set -e; mkdir -p $(SYFT_BIN); archive=$(SYFT_BIN)/syft.tar.gz; \
	curl -fsSL -o $$archive https://github.com/anchore/syft/releases/download/v$(SYFT_VERSION)/syft_$(SYFT_VERSION)_linux_amd64.tar.gz; \
	echo "$(SYFT_LINUX_AMD64_SHA256)  $$archive" | sha256sum -c - >&2; \
	tar -xzf $$archive -C $(SYFT_BIN) syft; rm $$archive; \
	echo $(SYFT_BIN)

$(GOVULNCHECK):
	GOBIN=$(GOVULNCHECK_BIN) go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

.PHONY: vuln
vuln: $(GOVULNCHECK)
	@set -e; for t in $(RELEASE_TARGETS); do \
		echo "govulncheck $$t (CGO_ENABLED=0)"; \
		GOOS=$${t%/*} GOARCH=$${t#*/} CGO_ENABLED=0 $(GOVULNCHECK) ./...; \
	done

# Linting targets
.PHONY: lint
lint:
	$(GOLANGCI_LINT) run --timeout 2m

.PHONY: fix-lint
fix-lint:
	@echo "Auto-fixing Go code formatting and imports..."
	@echo "1. Running gofmt to fix formatting..."
	@gofmt -w .
	@echo "2. Running goimports to fix imports..."
	@$(GOIMPORTS) -local $(LOCAL_PREFIX) -w .
	@echo "3. Running golangci-lint with --fix flag..."
	@$(GOLANGCI_LINT) run --fix --timeout 2m || true
	@echo ""
	@echo "Auto-fix complete! The following issues typically require manual fixing:"
	@echo "  - err113: Define static errors instead of dynamic ones"
	@echo "  - errcheck: Add error handling for function calls"
	@echo "  - revive: Fix exported function comments and unused parameters"
	@echo ""
	@echo "Run 'make lint' to see remaining issues that need manual attention."