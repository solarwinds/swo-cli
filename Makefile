golangci-lint-version = v1.60.3

.PHONY: install-golangci-lint
install-golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh >lint-install.sh
	mkdir -p bin
	chmod u+x ./lint-install.sh && ./lint-install.sh -b bin $(golangci-lint-version)
	$(RM) ./lint-install.sh

.PHONY: ci-lint
ci-lint: install-golangci-lint
	GOOS=linux bin/golangci-lint run --timeout 2m
	GOOS=windows bin/golangci-lint run --timeout 2m
	GOOS=darwin bin/golangci-lint run --timeout 2m
