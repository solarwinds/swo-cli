golangci-lint-version = v2.2.1

.PHONY: help
help: ## Show this help message
	@echo "Available targets:"
	@echo "  build          Build the swo binary"
	@echo "  test           Run all tests"
	@echo "  lint           Run linter and show issues"
	@echo "  fix-lint       Auto-fix formatting and some linting issues"
	@echo "  help           Show this help message"

# Build targets
.PHONY: build
build:
	go build -o swo ./cmd/swo
	chmod +x swo

.PHONY: test
test:
	go test ./...

# Linting targets
.PHONY: install-golangci-lint
install-golangci-lint:
	@if [ ! -f bin/golangci-lint ]; then \
		echo "Installing golangci-lint $(golangci-lint-version)..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh >lint-install.sh; \
		mkdir -p bin; \
		chmod u+x ./lint-install.sh && ./lint-install.sh -b bin $(golangci-lint-version); \
		$(RM) ./lint-install.sh; \
	else \
		echo "golangci-lint already installed in bin/"; \
	fi

.PHONY: lint
lint: install-golangci-lint
	bin/golangci-lint run --timeout 2m

.PHONY: fix-lint
fix-lint: install-golangci-lint
	@echo "Auto-fixing Go code formatting and imports..."
	@echo "1. Running gofmt to fix formatting..."
	@gofmt -w .
	@echo "2. Running goimports to fix imports..."
	@goimports -w .
	@echo "3. Running golangci-lint with --fix flag..."
	@bin/golangci-lint run --fix --timeout 2m || true
	@echo ""
	@echo "Auto-fix complete! The following issues typically require manual fixing:"
	@echo "  - err113: Define static errors instead of dynamic ones"
	@echo "  - errcheck: Add error handling for function calls"
	@echo "  - revive: Fix exported function comments and unused parameters"
	@echo ""
	@echo "Run 'make lint' to see remaining issues that need manual attention."