#   Copyright The containerd Authors.

#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at

#       http://www.apache.org/licenses/LICENSE-2.0

#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

# -----------------------------------------------------------------------------
# Portions from https://github.com/kubernetes-sigs/cri-tools/blob/v1.19.0/Makefile
# Copyright The Kubernetes Authors.
# Licensed under the Apache License, Version 2.0
# -----------------------------------------------------------------------------

DOCKER ?= docker
GO ?= go
GOOS ?= $(shell $(GO) env GOOS)
ifeq ($(GOOS),windows)
	BIN_EXT := .exe
endif

PACKAGE := github.com/containerd/nerdctl/v2

# distro builders might wanna override these
PREFIX  ?= /usr/local
BINDIR  ?= $(PREFIX)/bin
DATADIR ?= $(PREFIX)/share
DOCDIR  ?= $(DATADIR)/doc

GO_BUILD_LDFLAGS ?= -s -w
GO_BUILD_FLAGS ?=
export GO_BUILD=CGO_ENABLED=0 GOOS=$(GOOS) $(GO) -C $(MAKEFILE_DIR) build -ldflags "$(GO_BUILD_LDFLAGS) $(VERBOSE_FLAG) -X $(PACKAGE)/pkg/version.Version=$(VERSION) -X $(PACKAGE)/pkg/version.Revision=$(REVISION)"

# Variables
COMPANY_PREFIXES := "github.com/containerd"

MAKEFILE_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
VERSION ?= $(shell git -C $(MAKEFILE_DIR) describe --match 'v[0-9]*' --dirty='.m' --always --tags)
VERSION_TRIMMED := $(VERSION:v%=%)
REVISION ?= $(shell git -C $(MAKEFILE_DIR) rev-parse HEAD)$(shell if ! git -C $(MAKEFILE_DIR) diff --no-ext-diff --quiet --exit-code; then echo .m; fi)

ifdef VERBOSE
	VERBOSE_FLAG := -v
	VERBOSE_FLAG_LONG := --verbose
endif

LINT_COMMIT_RANGE ?= main..HEAD

ifndef DC_NO_FANCY
    NC := \033[0m
    GREEN := \033[1;32m
    ORANGE := \033[1;33m
    BLUE := \033[1;34m
    RED := \033[1;31m
endif

# Helpers
recursive_wildcard=$(wildcard $1$2) $(foreach e,$(wildcard $1*),$(call recursive_wildcard,$e/,$2))

define title
	@printf "$(GREEN)----------------------------------------------------------------------------------------------------\n"
	@printf "$(GREEN)%*s\n" $$(( ( $(shell echo "☆ $(1) ☆" | wc -c ) + 100 ) / 2 )) "☆ $(1) ☆"
	@printf "$(GREEN)----------------------------------------------------------------------------------------------------\n$(ORANGE)"
endef

define footer
	@printf "$(GREEN)> %s: done!\n" "$(1)"
	@printf "$(GREEN)____________________________________________________________________________________________________\n$(NC)"
endef

# Tasks
lint: lint-go-all lint-imports lint-yaml lint-shell lint-commits lint-headers lint-mod lint-licenses-all
test: test-unit race-unit bench-unit
unit: test-unit race-unit bench-unit
fix: fix-go fix-mod fix-imports

lint-go:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) && golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 --sort-results $(VERBOSE_FLAG_LONG) ./...
	$(call footer, $@)

#	&& GOOS=darwin make lint-go
lint-go-all:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& GOOS=linux make lint-go \
		&& GOOS=windows make lint-go
	$(call footer, $@)

lint-imports:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& ./hack/lint-imports.sh
	$(call footer, $@)

lint-yaml:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) && yamllint .
	$(call footer, $@)

lint-shell: $(call recursive_wildcard,$(MAKEFILE_DIR)/,*.sh)
	$(call title, $@)
	@shellcheck -a -x $^
	$(call footer, $@)

lint-commits:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) && git-validation $(VERBOSE_FLAG) -run DCO,short-subject,dangling-whitespace -range "$(LINT_COMMIT_RANGE)"
	$(call footer, $@)

lint-headers:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) && ltag -t "./hack/headers" --check $(VERBOSE_FLAG)
	$(call footer, $@)

lint-mod:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) && go mod tidy --diff
	$(call footer, $@)

# FIXME: go-licenses cannot find LICENSE from root of repo when submodule is imported:
# https://github.com/google/go-licenses/issues/186
# This is impacting gotest.tools
lint-licenses:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) && go-licenses check --include_tests --allowed_licenses=Apache-2.0,BSD-2-Clause,BSD-3-Clause,MIT \
	  --ignore gotest.tools \
	  --ignore github.com/containerd/stargz-snapshotter/estargz \
	  ./...
	@echo "WARNING: you need to manually verify licenses for:\n- gotest.tools\n- github.com/containerd/stargz-snapshotter/estargz"
	$(call footer, $@)

#	&& GOOS=darwin make lint-licenses
lint-licenses-all:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& GOOS=linux make lint-licenses \
		&& GOOS=windows make lint-licenses
	$(call footer, $@)

fix-go:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& golangci-lint run --fix
	$(call footer, $@)

fix-imports:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& goimports-reviser -company-prefixes $(COMPANY_PREFIXES) ./...
	$(call footer, $@)

fix-mod:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& go mod tidy
	$(call footer, $@)

up:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& go get -u ./...
	$(call footer, $@)

install-golangci:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& go install github.com/golangci/golangci-lint/cmd/golangci-lint@89476e7a1eaa0a8a06c17343af960a5fd9e7edb7 # v1.62.2
	$(call footer, $@)

install-linters:
	$(call title, $@)
	# git-validation: main from 2023/11
	# ltag: v0.2.5
	# go-licenses: v2.0.0-alpha.1
	# goimports-reviser: v3.8.2
	@cd $(MAKEFILE_DIR) \
		&& go install github.com/vbatts/git-validation@679e5cad8c50f1605ab3d8a0a947aaf72fb24c07 \
		&& go install github.com/kunalkushwaha/ltag@b0cfa33e4cc9383095dc584d3990b62c95096de0 \
		&& go install github.com/google/go-licenses/v2@d01822334fba5896920a060f762ea7ecdbd086e8 \
		&& go install github.com/incu6us/goimports-reviser/v3@f034195cc8a7ffc7cc70d60aa3a25500874eaf04
	$(call footer, $@)

test-unit:
	$(call title, $@)
	@go test $(VERBOSE_FLAG) -count 1 $(MAKEFILE_DIR)/pkg/...
	$(call footer, $@)

bench-unit:
	$(call title, $@)
	@go test $(VERBOSE_FLAG) -count 1 $(MAKEFILE_DIR)/pkg/... -bench=.
	$(call footer, $@)

race-unit:
	$(call title, $@)
	@go test $(VERBOSE_FLAG) -count 1 $(MAKEFILE_DIR)/pkg/... -race
	$(call footer, $@)

############################################
all: binaries

help:
	@echo "Usage: make <target>"
	@echo
	@echo " * 'install' - Install binaries to system locations."
	@echo " * 'binaries' - Build nerdctl."
	@echo " * 'clean' - Clean artifacts."
	@echo " * 'lint' - Run various linters."

nerdctl:
	$(GO_BUILD) $(GO_BUILD_FLAGS) $(VERBOSE_FLAG) -o $(CURDIR)/_output/nerdctl$(BIN_EXT) ./cmd/nerdctl

clean:
	find . -name \*~ -delete
	find . -name \#\* -delete
	rm -rf $(CURDIR)/_output/* $(MAKEFILE_DIR)/vendor

binaries: nerdctl

install:
	install -D -m 755 $(CURDIR)/_output/nerdctl $(DESTDIR)$(BINDIR)/nerdctl
	install -D -m 755 $(MAKEFILE_DIR)/extras/rootless/containerd-rootless.sh $(DESTDIR)$(BINDIR)/containerd-rootless.sh
	install -D -m 755 $(MAKEFILE_DIR)/extras/rootless/containerd-rootless-setuptool.sh $(DESTDIR)$(BINDIR)/containerd-rootless-setuptool.sh
	install -D -m 644 -t $(DESTDIR)$(DOCDIR)/nerdctl $(MAKEFILE_DIR)/docs/*.md

# Note that these options will not work on macOS - unless you use gnu-tar instead of tar
TAR_OWNER0_FLAGS=--owner=0 --group=0
TAR_FLATTEN_FLAGS=--transform 's/.*\///g'

define make_artifact_full_linux
	$(DOCKER) build --output type=tar,dest=$(CURDIR)/_output/nerdctl-full-$(VERSION_TRIMMED)-linux-$(1).tar --target out-full --platform $(1) --build-arg GO_VERSION -f $(MAKEFILE_DIR)/Dockerfile $(MAKEFILE_DIR)
	gzip -9 $(CURDIR)/_output/nerdctl-full-$(VERSION_TRIMMED)-linux-$(1).tar
endef

artifacts: clean
	GOOS=linux GOARCH=amd64       make -C $(CURDIR) -f $(MAKEFILE_DIR)/Makefile binaries
	tar $(TAR_OWNER0_FLAGS) $(TAR_FLATTEN_FLAGS) -czvf $(CURDIR)/_output/nerdctl-$(VERSION_TRIMMED)-linux-amd64.tar.gz   $(CURDIR)/_output/nerdctl $(MAKEFILE_DIR)/extras/rootless/*

	GOOS=linux GOARCH=arm64       make -C $(CURDIR) -f $(MAKEFILE_DIR)/Makefile binaries
	tar $(TAR_OWNER0_FLAGS) $(TAR_FLATTEN_FLAGS) -czvf $(CURDIR)/_output/nerdctl-$(VERSION_TRIMMED)-linux-arm64.tar.gz   $(CURDIR)/_output/nerdctl $(MAKEFILE_DIR)/extras/rootless/*

	GOOS=linux GOARCH=arm GOARM=7 make -C $(CURDIR) -f $(MAKEFILE_DIR)/Makefile binaries
	tar $(TAR_OWNER0_FLAGS) $(TAR_FLATTEN_FLAGS) -czvf $(CURDIR)/_output/nerdctl-$(VERSION_TRIMMED)-linux-arm-v7.tar.gz  $(CURDIR)/_output/nerdctl $(MAKEFILE_DIR)/extras/rootless/*

	GOOS=linux GOARCH=ppc64le     make -C $(CURDIR) -f $(MAKEFILE_DIR)/Makefile binaries
	tar $(TAR_OWNER0_FLAGS) $(TAR_FLATTEN_FLAGS) -czvf $(CURDIR)/_output/nerdctl-$(VERSION_TRIMMED)-linux-ppc64le.tar.gz $(CURDIR)/_output/nerdctl $(MAKEFILE_DIR)/extras/rootless/*

	GOOS=linux GOARCH=riscv64     make -C $(CURDIR) -f $(MAKEFILE_DIR)/Makefile binaries
	tar $(TAR_OWNER0_FLAGS) $(TAR_FLATTEN_FLAGS) -czvf $(CURDIR)/_output/nerdctl-$(VERSION_TRIMMED)-linux-riscv64.tar.gz $(CURDIR)/_output/nerdctl $(MAKEFILE_DIR)/extras/rootless/*

	GOOS=linux GOARCH=s390x       make -C $(CURDIR) -f $(MAKEFILE_DIR)/Makefile binaries
	tar $(TAR_OWNER0_FLAGS) $(TAR_FLATTEN_FLAGS) -czvf $(CURDIR)/_output/nerdctl-$(VERSION_TRIMMED)-linux-s390x.tar.gz   $(CURDIR)/_output/nerdctl $(MAKEFILE_DIR)/extras/rootless/*

	GOOS=windows GOARCH=amd64     make -C $(CURDIR) -f $(MAKEFILE_DIR)/Makefile binaries
	tar $(TAR_OWNER0_FLAGS) $(TAR_FLATTEN_FLAGS) -czvf $(CURDIR)/_output/nerdctl-$(VERSION_TRIMMED)-windows-amd64.tar.gz $(CURDIR)/_output/nerdctl.exe

	rm -f $(CURDIR)/_output/nerdctl $(CURDIR)/_output/nerdctl.exe

	$(call make_artifact_full_linux,amd64)
	$(call make_artifact_full_linux,arm64)

	$(GO) -C $(MAKEFILE_DIR) mod vendor
	tar $(TAR_OWNER0_FLAGS) -czf $(CURDIR)/_output/nerdctl-$(VERSION_TRIMMED)-go-mod-vendor.tar.gz $(MAKEFILE_DIR)/go.mod $(MAKEFILE_DIR)/go.sum $(MAKEFILE_DIR)/vendor

.PHONY: \
	help \
	nerdctl \
	clean \
	binaries \
	install \
	artifacts \
	\
	lint lint-commits lint-go lint-go-all lint-headers lint-imports lint-licenses lint-licenses-all lint-mod lint-shell lint-yaml \
	install-golangci install-linters \
	fix fix-go fix-imports fix-mod \
	update \
	test test-unit race-unit bench-unit \
	unit