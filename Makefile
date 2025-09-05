golangci-lint-version = v2.2.1

.PHONY: build
build:
	go build -o swo ./cmd/swo
	chmod +x swo

.PHONY: build
test:
	go test ./...

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