golangci-lint-version = v1.56.1
ifeq ($(GOOS),windows)
  WORKSPACE := ${shell go env GOPATH}
else
  WORKSPACE := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
endif

.PHONY: install-golangci-lint
install-golangci-lint:
	$(call print-target)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh >lint-install.sh
	mkdir $(WORKSPACE)/bin
	chmod u+x ./lint-install.sh && ./lint-install.sh -b $(WORKSPACE)/bin $(golangci-lint-version)
	$(RM) ./lint-install.sh

.PHONY: ci-lint
ci-lint: install-golangci-lint
	$(call print-target)
	GOOS=linux "$(WORKSPACE)/bin/golangci-lint" run --timeout 2m
	GOOS=windows "$(WORKSPACE)/bin/golangci-lint" run --timeout 2m
	GOOS=darwin "$(WORKSPACE)/bin/golangci-lint" run --timeout 2m
