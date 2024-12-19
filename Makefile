# Copyright 2024 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.DEFAULT_GOAL := help
.SUFFIXES: # remove legacy builtin suffixes to allow easier make debugging
SHELL = /usr/bin/env bash

# If GOARCH is not set in the env, find it
GOARCH ?= $(shell go env GOARCH)

## ==== ARGS =====
DOCKER            ?= docker                    # Container build tool compatible with `docker` API
PLATFORM          ?= linux/$(GOARCH)           # Platform for 'build'
BUILD_ARGS        ?=                           # Additional args for 'build'
SAMPLE_DRIVER_TAG ?= cosi-driver-sample:latest # Image tag for controller image build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: clean
clean:
	rm -rf $(LOCALBIN)

.PHONY: all
all: clean lint test

##@ Image

.PHONY: build
build: ## Build only the controller container image
	$(DOCKER) build --file Dockerfile --platform $(PLATFORM) $(BUILD_ARGS) --tag $(SAMPLE_DRIVER_TAG) .

##@ Release

.PHONY: release-tag
release-tag: ## Tag the repository for release.
	git tag $(GIT_TAG)
	git push origin $(GIT_TAG)

##@ Development

.PHONY: test
test: test-unit ## Run all tests.

.PHONY: test-unit
test-unit: ## Run unit tests.
	GO111MODULE=on GOARCH=$(ARCH) go test -cover -race ./pkg/... ./cmd/...

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter.
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes.
	$(GOLANGCI_LINT) run --fix

.PHONY: lint-logging
lint-logging: logcheck ## Run logcheck linter to verify the logging practices.
	$(LOGCHECK) ./... || (echo 'Fix structured logging' && exit 1)

.PHONY: lint-manifests
lint-manifests: kustomize kube-linter ## Run kube-linter on Kubernetes manifests.
	$(KUSTOMIZE) build config/default |\
		$(KUBE_LINTER) lint --config=./config/.kube-linter.yaml -

.PHONY: verify-licenses
verify-licenses: addlicense ## Run addlicense to verify if files have license headers.
	find -type f -name "*.go" ! -path "*/vendor/*" | xargs $(ADDLICENSE) -check || (echo 'Run "make update"' && exit 1)

.PHONY: add-licenses
add-licenses: addlicense ## Run addlicense to append license headers to files missing one.
	find -type f -name "*.go" ! -path "*/vendor/*" | xargs $(ADDLICENSE) -c "The Kubernetes Authors."

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
ADDLICENSE    ?= $(LOCALBIN)/addlicense
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
KUBE_LINTER   ?= $(LOCALBIN)/kube-linter
KUSTOMIZE     ?= $(LOCALBIN)/kustomize
LOGCHECK      ?= $(LOCALBIN)/logcheck

## Tool Versions
ADDLICENSE_VERSION    ?= $(shell grep 'github.com/google/addlicense '       ./go.mod | cut -d ' ' -f 2)
GOLANGCI_LINT_VERSION ?= $(shell grep 'github.com/golangci/golangci-lint '  ./go.mod | cut -d ' ' -f 2)
KUBE_LINTER_VERSION   ?= $(shell grep 'golang.stackrox.io/kube-linter '     ./go.mod | cut -d ' ' -f 2)
KUSTOMIZE_VERSION     ?= $(shell grep 'sigs.k8s.io/kustomize/kustomize/v5 ' ./go.mod | cut -d ' ' -f 2)
LOGCHECK_VERSION      ?= $(shell grep 'sigs.k8s.io/logtools '               ./go.mod | cut -d ' ' -f 2)

.PHONY: tools
tools: addlicense golangci-lint kube-linter kustomize logcheck

.PHONY: addlicense
addlicense: $(ADDLICENSE)$(ADDLICENSE_VERSION) ## Download addlicense locally if necessary.
$(ADDLICENSE)$(ADDLICENSE_VERSION): $(LOCALBIN)
	$(call go-install-tool,$(ADDLICENSE),github.com/google/addlicense,$(ADDLICENSE_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT)$(GOLANGCI_LINT_VERSION) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT)$(GOLANGCI_LINT_VERSION): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: kube-linter
kube-linter: $(KUBE_LINTER)$(KUBE_LINTER_VERSION)
$(KUBE_LINTER)$(KUBE_LINTER_VERSION): $(LOCALBIN) ## Download kube-linter locally if necessary.
	$(call go-install-tool,$(KUBE_LINTER),golang.stackrox.io/kube-linter/cmd/kube-linter,$(KUBE_LINTER_VERSION))

.PHONY: kustomize
kustomize: $(KUSTOMIZE)$(KUSTOMIZE_VERSION) ## Download kustomize locally if necessary.
$(KUSTOMIZE)$(KUSTOMIZE_VERSION): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: logcheck
logcheck: $(LOGCHECK)$(LOGCHECK_VERSION) ## Download logcheck locally if necessary.
$(LOGCHECK)$(LOGCHECK_VERSION): $(LOCALBIN)
	$(call go-install-tool,$(LOGCHECK),sigs.k8s.io/logtools/logcheck,$(LOGCHECK_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef
