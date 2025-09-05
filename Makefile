golangci-lint-version = v2.2.1

.PHONY: build
build:
	go build -o swo ./cmd/swo
	chmod +x swo

.PHONY: build
test:
	go test ./config ./logs

.PHONY: install-golangci-lint
install-golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh >lint-install.sh
	mkdir -p bin
	chmod u+x ./lint-install.sh && ./lint-install.sh -b bin $(golangci-lint-version)
	$(RM) ./lint-install.sh

.PHONY: ci-lint
ci-lint: install-golangci-lint
	bin/golangci-lint run --timeout 2m