# Copyright 2018 The Kubernetes Authors.
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

GO := go
PROJECT := k8s.io/frakti
BINDIR := /usr/local/bin
BUILD_DIR := _output
BUILD_TAGS := seccomp apparmor
# Add `-TEST` suffix to indicate that all binaries built from this repo are for test.
GO_LDFLAGS := -X $(PROJECT)/vendor/github.com/containerd/containerd/version.Version=1.1.0-TEST
VERSION := 0.1
SOURCES := $(shell find cmd/ pkg/ vendor/ -name '*.go')
PLUGIN_SOURCES := $(shell ls *.go)

.PHONY: all
all: binaries

.PHONY: help
help:
	@echo "Usage: make <target>"
	@echo
	@echo " * 'install'          	- Install binaries to system locations"
	@echo " * 'binaries'         	- Build containerd and ctr"
	@echo " * 'ctr'  		- Build ctr"
	@echo " * 'install-ctr' 	- Install ctr"
	@echo " * 'containerd'  	- Build a customized containerd with CRI plugin for testing"
	@echo " * 'install-containerd'	- Install customized containerd to system location"
	@echo " * 'clean'            	- Clean artifacts"
	@echo " * 'verify'           	- Execute the source code verification tools"
	@echo " * 'install.tools'    	- Install tools used by verify"
	@echo " * 'install.deps'     	- Install dependencies of cri (Note: BUILDTAGS defaults to 'seccomp apparmor' for runc build")
	@echo " * 'uninstall'        	- Remove installed binaries from system locations"
	@echo " * 'version'          	- Print current containerd-kata plugin version"
	@echo " * 'update-vendor'    	- Syncs containerd/vendor.conf -> vendor.conf and sorts vendor.conf"

.PHONY: verify
verify: lint gofmt boiler

.PHONY: version
version:
	@echo $(VERSION)

.PHONY: lint
lint:
	@echo "checking lint"
	@./hack/verify-lint.sh

.PHONY: gofmt
gofmt:
	@echo "checking gofmt"
	@./hack/verify-gofmt.sh

.PHONY: boiler
boiler:
	@echo "checking boilerplate"
	@./hack/verify-boilerplate.sh

.PHONY: $(BUILD_DIR)/ctr
$(BUILD_DIR)/ctr: $(SOURCES)
	$(GO) build -o $@ \
		-tags '$(BUILD_TAGS)' \
		-ldflags '$(GO_LDFLAGS)' \
		$(PROJECT)/cmd/ctr

.PHONY: $(BUILD_DIR)/containerd
$(BUILD_DIR)/containerd: $(SOURCES) $(PLUGIN_SOURCES)
	$(GO) build -o $@ \
		-tags '$(BUILD_TAGS)' \
		-ldflags '$(GO_LDFLAGS)' \
		$(PROJECT)/cmd/containerd

.PHONY: binaries
binaries: $(BUILD_DIR)/containerd $(BUILD_DIR)/ctr

.PHONY: ctr
ctr: $(BUILD_DIR)/ctr

.PHONY: install-ctr
install-ctr: ctr
	install -D -m 755 $(BUILD_DIR)/ctr $(BINDIR)/test-ctr

.PHONY: containerd
containerd: $(BUILD_DIR)/containerd

.PHONY: install-containerd
install-containerd: containerd
	install -D -m 755 $(BUILD_DIR)/containerd $(BINDIR)/test-containerd

.PHONY: install
install: install-ctr install-containerd

.PHONY: uninstall
uninstall:
	rm -f $(BINDIR)/test-containerd
	rm -f $(BINDIR)/test-ctr

.PHONY: clean 
clean:
	rm -rf $(BUILD_DIR)/*

.PHONY: install.tools 
install.tools: .install.gometalinter

.PHONY: .install.gometalinter
.install.gometalinter:
	$(GO) get -u github.com/alecthomas/gometalinter
	gometalinter --install
