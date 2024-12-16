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
BINARY := lepton

# distro builders might wanna override these
PREFIX  ?= /usr/local
BINDIR  ?= $(PREFIX)/bin
DATADIR ?= $(PREFIX)/share
DOCDIR  ?= $(DATADIR)/doc

MAKEFILE_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
VERSION ?= $(shell git -C $(MAKEFILE_DIR) describe --match 'v[0-9]*' --dirty='.m' --always --tags)
VERSION_TRIMMED := $(VERSION:v%=%)
REVISION ?= $(shell git -C $(MAKEFILE_DIR) rev-parse HEAD)$(shell if ! git -C $(MAKEFILE_DIR) diff --no-ext-diff --quiet --exit-code; then echo .m; fi)

ifdef VERBOSE
	VERBOSE_FLAG := -v
	VERBOSE_FLAG_LONG := --verbose
endif

ifndef LINT_COMMIT_RANGE
	LINT_COMMIT_RANGE := main..HEAD
endif

GO_BUILD_LDFLAGS ?= -s -w
GO_BUILD_FLAGS ?=
export GO_BUILD=CGO_ENABLED=0 GOOS=$(GOOS) $(GO) -C $(MAKEFILE_DIR) build -ldflags "$(GO_BUILD_LDFLAGS) $(VERBOSE_FLAG) -X $(PACKAGE)/pkg/version.RootName=$(BINARY) -X $(PACKAGE)/pkg/version.Version=$(VERSION) -X $(PACKAGE)/pkg/version.Revision=$(REVISION)"

recursive_wildcard=$(wildcard $1$2) $(foreach e,$(wildcard $1*),$(call recursive_wildcard,$e/,$2))

all: binaries

help:
	@echo "Usage: make <target>"
	@echo
	@echo " * 'install' - Install binaries to system locations."
	@echo " * 'binaries' - Build $(BINARY)."
	@echo " * 'clean' - Clean artifacts."
	@echo " * 'lint' - Run various linters."

$(BINARY):
	$(GO_BUILD) $(GO_BUILD_FLAGS) $(VERBOSE_FLAG) -o $(CURDIR)/_output/$(BINARY)$(BIN_EXT) ./cmd/$(BINARY)

clean:
	find . -name \*~ -delete
	find . -name \#\* -delete
	rm -rf $(CURDIR)/_output/* $(MAKEFILE_DIR)/vendor

lint-install-golangci:
	@cd $(MAKEFILE_DIR) \
		&& go install github.com/golangci/golangci-lint/cmd/golangci-lint@89476e7a1eaa0a8a06c17343af960a5fd9e7edb7 # v1.62.2

lint-install-tools:
	# git-validation: main from 2023/11
	# ltag: v0.2.5
	# go-licenses: v2.0.0-alpha.1
	# goimports-reviser: v3.8.2
	@cd $(MAKEFILE_DIR) \
		&& go install github.com/vbatts/git-validation@679e5cad8c50f1605ab3d8a0a947aaf72fb24c07 \
		&& go install github.com/kunalkushwaha/ltag@b0cfa33e4cc9383095dc584d3990b62c95096de0 \
		&& go install github.com/google/go-licenses/v2@d01822334fba5896920a060f762ea7ecdbd086e8 \
		&& go install github.com/incu6us/goimports-reviser/v3@f034195cc8a7ffc7cc70d60aa3a25500874eaf04

lint-fix-imports:
	cd $(MAKEFILE_DIR) && goimports-reviser -company-prefixes "github.com/containerd" ./...

lint: lint-go-all lint-imports lint-yaml lint-shell lint-commits lint-headers lint-mod lint-licenses-all

lint-go-all:
	@cd $(MAKEFILE_DIR) \
		&& GOOS=linux golangci-lint run $(VERBOSE_FLAG_LONG) ./... \
		&& GOOS=windows golangci-lint run $(VERBOSE_FLAG_LONG) ./...

lint-go:
	@cd $(MAKEFILE_DIR) && golangci-lint run $(VERBOSE_FLAG_LONG) ./...

lint-imports:
	@cd $(MAKEFILE_DIR) && ./hack/lint-imports.sh

lint-yaml:
	@cd $(MAKEFILE_DIR) && yamllint .

lint-shell: $(call recursive_wildcard,$(MAKEFILE_DIR)/,*.sh)
	@shellcheck -a -x $^

lint-commits:
	@cd $(MAKEFILE_DIR) && git-validation -v -run DCO,short-subject,dangling-whitespace -range "$(LINT_COMMIT_RANGE)"

lint-headers:
	@cd $(MAKEFILE_DIR) && ltag -t "./hack/headers" --check -v

lint-mod:
	@cd $(MAKEFILE_DIR) && go mod tidy --diff

lint-licenses-all:
	@cd $(MAKEFILE_DIR) \
		&& GOOS=linux make lint-licenses \
		&& GOOS=windows make lint-licenses

# FIXME: go-licenses cannot find LICENSE from root of repo when submodule is imported:
# https://github.com/google/go-licenses/issues/186
# This is impacting gotest.tools and estargz
lint-licenses:
	@cd $(MAKEFILE_DIR) && go-licenses check --include_tests --allowed_licenses=Apache-2.0,BSD-2-Clause,BSD-3-Clause,MIT \
	  --ignore gotest.tools \
	  --ignore github.com/containerd/stargz-snapshotter/estargz \
	  ./...
	@echo "WARNING: you need to manually verify licenses for:\n- gotest.tools\n- github.com/containerd/stargz-snapshotter/estargz"

test-unit:
	go test -v $(MAKEFILE_DIR)/pkg/...

binaries: $(BINARY)

install:
	install -D -m 755 $(CURDIR)/_output/$(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)
	install -D -m 755 $(MAKEFILE_DIR)/extras/rootless/containerd-rootless.sh $(DESTDIR)$(BINDIR)/containerd-rootless.sh
	install -D -m 755 $(MAKEFILE_DIR)/extras/rootless/containerd-rootless-setuptool.sh $(DESTDIR)$(BINDIR)/containerd-rootless-setuptool.sh
	install -D -m 644 -t $(DESTDIR)$(DOCDIR)/$(BINARY) $(MAKEFILE_DIR)/docs/*.md

# Note that these options will not work on macOS - unless you use gnu-tar instead of tar
TAR_OWNER0_FLAGS=--owner=0 --group=0
TAR_FLATTEN_FLAGS=--transform 's/.*\///g'

define make_artifact_full_linux
	$(DOCKER) build --output type=tar,dest=$(CURDIR)/_output/$(BINARY)-full-$(VERSION_TRIMMED)-linux-$(1).tar --target out-full --platform $(1) --build-arg GO_VERSION -f $(MAKEFILE_DIR)/Dockerfile $(MAKEFILE_DIR)
	gzip -9 $(CURDIR)/_output/$(BINARY)-full-$(VERSION_TRIMMED)-linux-$(1).tar
endef

artifacts: clean
	GOOS=linux GOARCH=amd64       make -C $(CURDIR) -f $(MAKEFILE_DIR)/Makefile binaries
	tar $(TAR_OWNER0_FLAGS) $(TAR_FLATTEN_FLAGS) -czvf $(CURDIR)/_output/$(BINARY)-$(VERSION_TRIMMED)-linux-amd64.tar.gz   $(CURDIR)/_output/$(BINARY) $(MAKEFILE_DIR)/extras/rootless/*

	GOOS=linux GOARCH=arm64       make -C $(CURDIR) -f $(MAKEFILE_DIR)/Makefile binaries
	tar $(TAR_OWNER0_FLAGS) $(TAR_FLATTEN_FLAGS) -czvf $(CURDIR)/_output/$(BINARY)-$(VERSION_TRIMMED)-linux-arm64.tar.gz   $(CURDIR)/_output/$(BINARY) $(MAKEFILE_DIR)/extras/rootless/*

	GOOS=linux GOARCH=arm GOARM=7 make -C $(CURDIR) -f $(MAKEFILE_DIR)/Makefile binaries
	tar $(TAR_OWNER0_FLAGS) $(TAR_FLATTEN_FLAGS) -czvf $(CURDIR)/_output/$(BINARY)-$(VERSION_TRIMMED)-linux-arm-v7.tar.gz  $(CURDIR)/_output/$(BINARY) $(MAKEFILE_DIR)/extras/rootless/*

	GOOS=linux GOARCH=ppc64le     make -C $(CURDIR) -f $(MAKEFILE_DIR)/Makefile binaries
	tar $(TAR_OWNER0_FLAGS) $(TAR_FLATTEN_FLAGS) -czvf $(CURDIR)/_output/$(BINARY)-$(VERSION_TRIMMED)-linux-ppc64le.tar.gz $(CURDIR)/_output/$(BINARY) $(MAKEFILE_DIR)/extras/rootless/*

	GOOS=linux GOARCH=riscv64     make -C $(CURDIR) -f $(MAKEFILE_DIR)/Makefile binaries
	tar $(TAR_OWNER0_FLAGS) $(TAR_FLATTEN_FLAGS) -czvf $(CURDIR)/_output/$(BINARY)-$(VERSION_TRIMMED)-linux-riscv64.tar.gz $(CURDIR)/_output/$(BINARY) $(MAKEFILE_DIR)/extras/rootless/*

	GOOS=linux GOARCH=s390x       make -C $(CURDIR) -f $(MAKEFILE_DIR)/Makefile binaries
	tar $(TAR_OWNER0_FLAGS) $(TAR_FLATTEN_FLAGS) -czvf $(CURDIR)/_output/$(BINARY)-$(VERSION_TRIMMED)-linux-s390x.tar.gz   $(CURDIR)/_output/$(BINARY) $(MAKEFILE_DIR)/extras/rootless/*

	GOOS=windows GOARCH=amd64     make -C $(CURDIR) -f $(MAKEFILE_DIR)/Makefile binaries
	tar $(TAR_OWNER0_FLAGS) $(TAR_FLATTEN_FLAGS) -czvf $(CURDIR)/_output/$(BINARY)-$(VERSION_TRIMMED)-windows-amd64.tar.gz $(CURDIR)/_output/$(BINARY).exe

	rm -f $(CURDIR)/_output/$(BINARY) $(CURDIR)/_output/$(BINARY).exe

	$(call make_artifact_full_linux,amd64)
	$(call make_artifact_full_linux,arm64)

	$(GO) -C $(MAKEFILE_DIR) mod vendor
	tar $(TAR_OWNER0_FLAGS) -czf $(CURDIR)/_output/$(BINARY)-$(VERSION_TRIMMED)-go-mod-vendor.tar.gz $(MAKEFILE_DIR)/go.mod $(MAKEFILE_DIR)/go.sum $(MAKEFILE_DIR)/vendor

.PHONY: \
	help \
	clean \
	$(BINARY) \
	binaries \
	install \
	artifacts
	lint \
	lint-yaml \
	lint-go \
	lint-shell
