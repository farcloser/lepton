#   Copyright Farcloser.

#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at

#       http://www.apache.org/licenses/LICENSE-2.0

#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

# FIXME: right now, this makefile is not working properly with mingw.
# For some reasons, the path gets mangled and make concept of path fails resolving installed binaries.
# Only certain tasks are usable - others will warn - others will just fail.

##########################
# Configuration
##########################
PACKAGE := "go.farcloser.world/lepton"
ORG_PREFIXES := "go.farcloser.world"
ICON := "⚛️"

DOCKER ?= docker
GO ?= go
GOOS ?= $(shell cd . && $(GO) env GOOS)
ifeq ($(GOOS),windows)
	BIN_EXT := .exe
endif

# distro builders might wanna override these
PREFIX  ?= /usr/local
BINDIR  ?= $(PREFIX)/bin
DATADIR ?= $(PREFIX)/share
DOCDIR  ?= $(DATADIR)/doc

BINARY ?= "lepton"
MAKEFILE_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
VERSION ?= $(shell git -C $(MAKEFILE_DIR) describe --match 'v[0-9]*' --dirty='.m' --always --tags 2>/dev/null \
	|| echo "no_git_information")
VERSION_TRIMMED := $(VERSION:v%=%)
REVISION ?= $(shell git -C $(MAKEFILE_DIR) rev-parse HEAD 2>/dev/null || echo "no_git_information")$(shell \
	if ! git -C $(MAKEFILE_DIR) diff --no-ext-diff --quiet --exit-code 2>/dev/null; then echo .m; fi)
LINT_COMMIT_RANGE ?= main..HEAD
GO_BUILD_LDFLAGS ?= -s -w
GO_BUILD_FLAGS ?=

##########################
# Helpers
##########################
ARCH := $(shell uname -m | sed -E "s/aarch64/arm64/")
ifneq ($(ARCH), arm64)
	ARCH = amd64
endif
OS := $(shell uname -s | tr '[:upper:]' '[:lower]')
ifneq ($(OS), darwin)
	ifneq ($(OS), linux)
		ifneq ($(OS), freebsd)
			OS = windows
		endif
	endif
endif

ifdef VERBOSE
	VERBOSE_FLAG := -v
	VERBOSE_FLAG_LONG := --verbose
endif

export GO_BUILD=CGO_ENABLED=0 GOOS=$(GOOS) $(GO) -C $(MAKEFILE_DIR) build -ldflags "$(GO_BUILD_LDFLAGS) $(VERBOSE_FLAG) -X $(PACKAGE)/pkg/version.Version=$(VERSION) -X $(PACKAGE)/pkg/version.Revision=$(REVISION) -X $(PACKAGE)/pkg/version.RootName=$(BINARY)"

ifndef NO_COLOR
    NC := \033[0m
    GREEN := \033[1;32m
    ORANGE := \033[1;33m
    BLUE := \033[1;34m
    RED := \033[1;31m
endif

recursive_wildcard=$(wildcard $1$2) $(foreach e,$(wildcard $1*),$(call recursive_wildcard,$e/,$2))

define title
	@printf "$(GREEN)____________________________________________________________________________________________________\n" 1>&2
	@printf "$(GREEN)%*s\n" $$(( ( $(shell echo "$(ICON)$(1) $(ICON)" | wc -c ) + 100 ) / 2 )) "$(ICON)$(1) $(ICON)" 1>&2
	@printf "$(GREEN)____________________________________________________________________________________________________\n$(ORANGE)" 1>&2
endef

define footer
	@printf "$(GREEN)> %s: done!\n" "$(1)" 1>&2
	@printf "$(GREEN)____________________________________________________________________________________________________\n$(NC)" 1>&2
endef

REMAKE := make -C $(CURDIR) -f $(MAKEFILE_DIR)/Makefile

##########################
# High-level tasks definitions
##########################

# Tasks
lint: lint-go-all lint-commits lint-mod lint-licenses-all lint-headers lint-shell lint-yaml

fix: fix-mod fix-go-all

test: unit

unit: test-unit test-unit-race test-unit-bench

##########################
# Linting tasks
##########################
lint-go:
	$(call title, $@: $(GOOS))
	@cd $(MAKEFILE_DIR) \
		&& golangci-lint run $(VERBOSE_FLAG_LONG) ./...
	$(call footer, $@)

lint-go-all:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& GOOS=linux make lint-go \
		&& GOOS=windows make lint-go
	$(call footer, $@)

lint-yaml:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& yamllint .
	$(call footer, $@)

lint-shell: $(call recursive_wildcard,$(MAKEFILE_DIR)/,*.sh)
	$(call title, $@)
	@shellcheck -a -x $^
	$(call footer, $@)

# See https://github.com/andyfeller/gh-ssh-allowed-signers for automation to retrieve contributors keys
lint-commits:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& git config --add gpg.ssh.allowedSignersFile hack/allowed_signers \
		&& git-validation $(VERBOSE_FLAG) -run DCO,short-subject,dangling-whitespace -range "$(LINT_COMMIT_RANGE)"
	$(call footer, $@)

lint-headers:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& ltag -t "./hack/headers" --check -v
	$(call footer, $@)

lint-mod:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& go mod tidy --diff
	$(call footer, $@)

# FIXME: go-licenses cannot find LICENSE from root of repo when submodule is imported:
# https://github.com/google/go-licenses/issues/186
# This is impacting gotest.tools
# FIXME: go-base36 is multi-license (MIT/Apache), using a custom boilerplate file that go-licenses fails to understand
lint-licenses:
	$(call title, $@: $(GOOS))
	@cd $(MAKEFILE_DIR) \
		&& go-licenses check --include_tests --allowed_licenses=Apache-2.0,BSD-2-Clause,BSD-3-Clause,MIT,MPL-2.0 \
		  --ignore gotest.tools \
		  --ignore github.com/multiformats/go-base36 \
		  ./...
	$(call footer, $@)

lint-licenses-all:
	$(call title, $@)
	cd $(MAKEFILE_DIR) \
		&& GOOS=linux make lint-licenses \
		&& GOOS=windows make lint-licenses
	$(call footer, $@)

##########################
# Automated fixing tasks
##########################
fix-go:
	$(call title, $@: $(GOOS))
	@cd $(MAKEFILE_DIR) \
		&& golangci-lint run --fix
	$(call footer, $@)

fix-go-all:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& GOOS=linux make fix-go \
		&& GOOS=windows make fix-go
	$(call footer, $@)

fix-mod:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& go mod tidy
	$(call footer, $@)

fix-up:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& go get -u ./...
	$(call footer, $@)

##########################
# Development tools installation
##########################
install-dev-gotestsum:
	# gotestsum: 1.12.1 (2025-03-15)
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& go install gotest.tools/gotestsum@3f7ff0ec4aeb6f95f5d67c998b71f272aa8a8b41
	$(call footer, $@)

install-dev-tools: install-dev-gotestsum
	$(call title, $@)
	# golangci: v2.0.2 (2024-03-26)
	# git-validation: main (2025-02-25)
	# ltag: main (2025-03-04)
	# go-licenses: v2.0.0-alpha.1 (2024-06-27)
	# kind: v0.27.0 (2025-02-14)
	# FIXME: remove kind - nothing to do here
	@cd $(MAKEFILE_DIR) \
		&& go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@2b224c2cf4c9f261c22a16af7f8ca6408467f338 \
		&& go install github.com/vbatts/git-validation@7b60e35b055dd2eab5844202ffffad51d9c93922 \
		&& go install github.com/containerd/ltag@66e6a514664ee2d11a470735519fa22b1a9eaabd \
		&& go install github.com/google/go-licenses/v2@d01822334fba5896920a060f762ea7ecdbd086e8 \
		&& go install sigs.k8s.io/kind@6cb934219ac54aa0ddb1d8313adc05304421ccb6
	@echo "Remember to add \$$HOME/go/bin to your path"
	$(call footer, $@)

GO_VERSION ?= stable
GO_VERSION_SELECTOR = .version | startswith("go$(GO_VERSION)")
ifeq ($(GO_VERSION),canary)
	GO_VERSION_SELECTOR = .stable==false
endif
ifeq ($(GO_VERSION),stable)
	GO_VERSION_SELECTOR = .stable==true
endif
ifeq ($(GO_VERSION),)
	GO_VERSION_SELECTOR = .stable==true
endif

GO_INSTALL_DESTINATION ?= /opt/$(BINARY)-dev-tools

install-go:
	$(call title, $@)
	@mkdir -p $(GO_INSTALL_DESTINATION)
	@if [ ! -e $(GO_INSTALL_DESTINATION)/go ]; then cd $(GO_INSTALL_DESTINATION); \
		curl -o go.archive -fsSL --proto '=https' --tlsv1.2 https://go.dev/dl/$(shell \
			curl --proto "=https" --tlsv1.2 -fsSL "https://go.dev/dl/?mode=json&include=all" | \
			jq -rc 'map(select($(GO_VERSION_SELECTOR)))[0].files | map(select(.os=="$(OS)" and .arch=="$(ARCH)"))[0].filename'); \
		[ "$(OS)" = windows ] && unzip go.archive >/dev/null || tar xzf go.archive; \
	else \
		echo "Install already detected in $(GO_INSTALL_DESTINATION), doing nothing."; \
	fi
	@echo Remember to add to your profile: export PATH="$(GO_INSTALL_DESTINATION)/go/bin:\$$HOME/go/bin:\$$PATH"
	$(call footer, $@)

install-go-resolve-version:
	@curl --proto "=https" --tlsv1.2 -fsSL "https://go.dev/dl/?mode=json&include=all" | \
		jq -rc 'map(select($(GO_VERSION_SELECTOR)))[0].version' | sed s/go//

test-unit:
	$(call title, $@)
	@cd . && go test $(VERBOSE_FLAG) -count 1 $(MAKEFILE_DIR)/pkg/...
	$(call footer, $@)

test-unit-bench:
	$(call title, $@)
	@cd . && go test $(VERBOSE_FLAG) -count 1 $(MAKEFILE_DIR)/pkg/... -bench=.
	$(call footer, $@)

test-unit-race:
	$(call title, $@)
	@cd . && CGO_ENABLED=1 go test $(VERBOSE_FLAG) $(MAKEFILE_DIR)/pkg/... -race
	$(call footer, $@)

##########################
# Building tasks
##########################
$(BINARY):
	$(call title, $@: $(GOOS) $(GOARCH))
	$(GO_BUILD) $(GO_BUILD_FLAGS) $(VERBOSE_FLAG) -o $(CURDIR)/_output/$(BINARY)$(BIN_EXT) ./cmd/$(BINARY)
	$(call footer, $@)

build: $(BINARY)

build-all:
	GOOS=linux GOARCH=amd64 $(REMAKE) build
	GOOS=linux GOARCH=arm64 $(REMAKE) build
	GOOS=linux GOARCH=riscv64 $(REMAKE) build
	GOOS=windows GOARCH=amd64 $(REMAKE) build
	GOOS=windows GOARCH=arm64 $(REMAKE) build

# FIXME: revisit helper scripts
# $^
# $(MAKEFILE_DIR)/extras/rootless/*
pack: build
	$(call title, $@: $(GOOS) $(GOARCH))
	tar --zstd -cf $(VERBOSE_FLAG) $(CURDIR)/_output/$(BINARY)-$(VERSION_TRIMMED)-$(GOOS)-$(GOARCH).tar.gz $(CURDIR)/_output/$(BINARY)$(BIN_EXT)
	$(call footer, $@)

pack-all:
	GOOS=linux GOARCH=amd64 $(REMAKE) pack
	GOOS=linux GOARCH=arm64 $(REMAKE) pack
	GOOS=linux GOARCH=riscv64 $(REMAKE) pack
	GOOS=windows GOARCH=amd64 $(REMAKE) pack
	GOOS=windows GOARCH=arm64 $(REMAKE) pack

clean:
	$(call title, $@)
	rm -rf $(CURDIR)/_output/* $(MAKEFILE_DIR)/vendor
	$(call footer, $@)

install:
	install -D -m 755 $(CURDIR)/_output/$(BINARY)$(BIN_EXT) $(DESTDIR)$(BINDIR)/$(BINARY)$(BIN_EXT)
	install -D -m 755 $(MAKEFILE_DIR)/extras/rootless/containerd-rootless.sh $(DESTDIR)$(BINDIR)/containerd-rootless.sh
	install -D -m 755 $(MAKEFILE_DIR)/extras/rootless/containerd-rootless-setuptool.sh $(DESTDIR)$(BINDIR)/containerd-rootless-setuptool.sh

.PHONY: \
	lint-go lint-go-all lint-yaml lint-shell lint-commits lint-headers lint-mod lint-licenses lint-licenses-all \
	fix-go fix-go-all fix-mod fix-up \
	install-dev-gotestsum install-dev-tools install-go \
	test-unit test-unit-bench test-unit-race \
	$(BINARY) \
	build pack install lint fix test unit clean \
	build-all pack-all

#	install -D -m 644 -t $(DESTDIR)$(DOCDIR)/$(BINARY) $(MAKEFILE_DIR)/docs/*.md
#define make_artifact_full_linux
#	$(DOCKER) build --output type=tar,dest=$(CURDIR)/_output/$(BINARY)-full-$(VERSION_TRIMMED)-linux-$(1).tar --target out-full --platform $(GOARCH) -f $(MAKEFILE_DIR)/Dockerfile $(MAKEFILE_DIR)
#	gzip -9 $(CURDIR)/_output/$(BINARY)-full-$(VERSION_TRIMMED)-linux-$(1).tar
#endef
#artifacts: build-all
#	$(call make_artifact_full_linux,amd64)
#	$(call make_artifact_full_linux,arm64)
#
#	$(GO) -C $(MAKEFILE_DIR) mod vendor

BUILDKIT_CACHE_COMPRESSION ?= zstd
BUILDKIT_CACHE_KEY ?= default
BUILDKIT_CACHE_LOCATION ?= $(HOME)/$(BINARY)-bk-cache-$(BUILDKIT_CACHE_KEY)-$(BUILDKIT_CACHE_COMPRESSION)
BUILDKIT_CACHE_FROM ?= type=local,src="$(BUILDKIT_CACHE_LOCATION)"
BUILDKIT_CACHE_TO ?= type=local,dest="$(BUILDKIT_CACHE_LOCATION)",compression=$(BUILDKIT_CACHE_COMPRESSION),mode=max
UBUNTU_VERSION ?= 24.04
CONTAINERD_VERSION ?= v2.0.4
BUILDKIT_IMAGE ?= moby/buildkit:v0.20.0
BUILDKIT_PLATFORM ?= linux/$(ARCH)
BUILDKIT_TARGET ?= assembly-runtime


# CI data:
# Warm:
# - buildctl no mount (12m53s 11m27s 11m31s)
#  - uncompressed, cache-export: 173.5s
#  - zstd, cache-export: 160.2s
#  - gzip, cache-export: 158.5s
# - buildctl mount (12m6s 12m46s 11m36s)
#  - uncompressed, cache-export: 160.6s
#  - zstd, cache-export: 173.7s
#  - gzip, cache-export: 164.9s
# - docker (12m7s 12m41s 12m40s)
#  - uncompressed, cache-export: 170.6s
#  - zstd, cache-export: 206.5s
#  - gzip, cache-export: 226.9s
build-image-target:
	$(call title, $@: $(BUILDKIT_TARGET) $(BUILDKIT_PLATFORM))
	@$(DOCKER) inspect $(BINARY)-make-builder 1>/dev/null 2>&1 || \
		$(DOCKER) run -d -v $(BUILDKIT_CACHE_LOCATION):$(BUILDKIT_CACHE_LOCATION) -v $(shell pwd):/src --name $(BINARY)-make-builder --privileged \
			--env ACTIONS_CACHE_URL=$(ACTIONS_CACHE_URL) \
			--env ACTIONS_RUNTIME_TOKEN=$(ACTIONS_RUNTIME_TOKEN) \
			$(BUILDKIT_IMAGE)
	@$(DOCKER) exec $(BINARY)-make-builder sh -c -- 'cd /src; buildctl build \\\
		--opt build-arg:UBUNTU_VERSION="$(UBUNTU_VERSION)" \\\
		--opt build-arg:CONTAINERD_VERSION="$(CONTAINERD_VERSION)" \\\
		--opt platform="$(BUILDKIT_PLATFORM)" \\\
		--import-cache="$(BUILDKIT_CACHE_FROM)" \\\
		--export-cache="$(BUILDKIT_CACHE_TO)" \\\
		--opt target="$(BUILDKIT_TARGET)" \\\
		--output type=docker,dest=$(pwd)/out.tar \\\
		--frontend=dockerfile.v0 \\\
		--local dockerfile=. \\\
		--local context=. \\\
'
	$(call footer, $@)

build-image-target-no-cache:
	$(call title, $@: $(BUILDKIT_TARGET) $(BUILDKIT_PLATFORM))
	@$(DOCKER) inspect $(BINARY)-make-builder 1>/dev/null 2>&1 || \
		$(DOCKER) run -d -v $(BUILDKIT_CACHE_LOCATION):$(BUILDKIT_CACHE_LOCATION) -v $(shell pwd):/src --name $(BINARY)-make-builder --privileged \
			--env ACTIONS_CACHE_URL=$(ACTIONS_CACHE_URL) \
			--env ACTIONS_RUNTIME_TOKEN=$(ACTIONS_RUNTIME_TOKEN) \
			$(BUILDKIT_IMAGE)
	@$(DOCKER) exec $(BINARY)-make-builder sh -c -- 'cd /src; buildctl build \\\
		--opt build-arg:UBUNTU_VERSION="$(UBUNTU_VERSION)" \\\
		--opt build-arg:CONTAINERD_VERSION="$(CONTAINERD_VERSION)" \\\
		--opt platform="$(BUILDKIT_PLATFORM)" \\\
		--opt target="$(BUILDKIT_TARGET)" \\\
		--output type=docker,dest=$(pwd)/out.tar \\\
		--frontend=dockerfile.v0 \\\
		--local dockerfile=. \\\
		--local context=. \\\
'
	$(call footer, $@)

# oci: 35s export - docker import FOREVER
# docker:
# - 35s export
# - docker import 0m42.600s
# - nerdctl import: 0m23.015s
