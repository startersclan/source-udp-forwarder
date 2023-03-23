# Copyright 2016 The Kubernetes Authors.
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

# The binary to build (just the basename).
BIN := $(shell basename $$PWD)

GOOS ?= linux
GOARCH ?= amd64

# Turn on / off go modules.
GO111MODULE = on

# Specify GOFLAGS. E.g. "-mod=vendor"
GOFLAGS =

# Where to push the docker image.
REGISTRY ?= docker.io
REGISTRY_USER ?= startersclan

###
### These variables should not need tweaking.
###

# This version-strategy uses git tags to set the version string
# Get the following from left to right: tag > branch > branch of detached HEAD commit
VERSION = $(shell (git describe --tags --exact-match 2>/dev/null || git symbolic-ref -q --short HEAD 2>/dev/null || git name-rev --name-only "$$( git rev-parse --short HEAD )" | sed 's@.*/@@') | tr '/' '-' | head -c10)

# Get the short SHA
SHA_SHORT = $(shell git rev-parse --short HEAD)

SRC_DIRS := cmd pkg # directories which hold app source (not vendored)

ALL_PLATFORMS := linux/amd64 linux/arm linux/arm64 linux/ppc64le linux/s390x

# Used internally.  Users should pass GOOS and/or GOARCH.
OS := $(if $(GOOS),$(GOOS),$(shell go env GOOS))
ARCH := $(if $(GOARCH),$(GOARCH),$(shell go env GOARCH))

# BASEIMAGE ?= gcr.io/distroless/static

IMAGE ?= $(REGISTRY)/$(REGISTRY_USER)/$(BIN)
TAG_SUFFIX := $(OS)-$(ARCH)

BUILD_IMAGE ?= golang:1.20

PWD := $$PWD

# Build directories
BUILD_GOPATH := .go
BUILD_GOCACHE := .go/.cache
BUILD_BIN_DIR := .go/bin
BUILD_DIR := build

# Directories that we need created to build/test.
BUILD_DIRS := $(BUILD_GOPATH) \
			  $(BUILD_GOCACHE) \
			  $(BUILD_BIN_DIR) \

OUTBIN = $(BUILD_BIN_DIR)/$(BIN)_$(VERSION)_$(OS)_$(ARCH)

COVERAGE_FILE ?= $(BUILD_GOPATH)/coverage.txt

$(BUILD_DIRS):
	@mkdir -p $@

# If you want to build all binaries, see the 'all-build' rule.
all: build

# For the following OS/ARCH expansions, we transform OS/ARCH into OS_ARCH
# because make pattern rules don't match with embedded '/' characters.

build-%:
	@$(MAKE) build \
		--no-print-directory \
		GOOS=$(firstword $(subst _, ,$*)) \
		GOARCH=$(lastword $(subst _, ,$*))

build-image-%:
	@$(MAKE) build-image \
		--no-print-directory \
		GOOS=$(firstword $(subst _, ,$*)) \
		GOARCH=$(lastword $(subst _, ,$*))

all-build: $(addprefix build-, $(subst /,_, $(ALL_PLATFORMS)))

build: $(OUTBIN)

# This will build the binary under ./.go
$(OUTBIN): | $(BUILD_DIRS)
	@echo "making $(OUTBIN)"
	@docker run \
		-i \
		--rm \
		-u $$(id -u):$$(id -g) \
		-v $(PWD):$(PWD) \
		-w $(PWD) \
		-v $(PWD)/$(BUILD_GOPATH):/go \
		-v $(PWD)/$(BUILD_GOCACHE):/.cache \
		--env HTTP_PROXY=$(HTTP_PROXY) \
		--env HTTPS_PROXY=$(HTTPS_PROXY) \
		$(BUILD_IMAGE) \
		/bin/sh -c " \
			ARCH=$(ARCH) \
			OS=$(OS) \
			GO111MODULE=$(GO111MODULE) \
			GOFLAGS=$(GOFLAGS) \
			OUTBIN=$(OUTBIN) \
			VERSION=$(VERSION) \
			COMMIT_SHA1=$(SHA_SHORT) \
			BUILD_DATE=$(shell date -u '+%Y-%m-%dT%H:%M:%S%z') \
			./build/build.sh \
		";

BUILDX_NAME := $(shell basename $$(pwd))
BUILDX_TAG_LATEST ?= false
BUILDX_PUSH ?= false
BUILDX_ARGS = \
	--progress plain \
	--cache-from type=local,src=/tmp/.buildx-cache \
	--cache-to type=local,dest=/tmp/.buildx-cache,mode=max \
	--build-arg "BUILD_IMAGE=$(BUILD_IMAGE)" \
	--build-arg "BUILD_DIR=$(BUILD_DIR)" \
	--build-arg "BUILD_BIN_DIR=$(BUILD_BIN_DIR)" \
	--build-arg "ARCH=$(ARCH)" \
	--build-arg "OS=$(OS)" \
	--build-arg "GO111MODULE=$(GO111MODULE)" \
	--build-arg "GOFLAGS=$(GOFLAGS)" \
	--build-arg "OUTBIN=$(OUTBIN)" \
	--build-arg "VERSION=$(VERSION)" \
	--build-arg "COMMIT_SHA1=$(SHA_SHORT)" \
	--build-arg "BUILD_DATE=$(shell date -u '+%Y-%m-%dT%H:%M:%S%z')" \
	--build-arg "PWD=$(PWD)" \
	--label OS=$(OS) \
	--label ARCH=$(ARCH) \
	--label VERSION=$(VERSION) \
	--label COMMIT_SHA1=$(COMMIT_SHA1) \
	--label BUILD_DATE=$(BUILD_DATE) \
	--tag "$(IMAGE):$(VERSION)" \
	--tag "$(IMAGE):$(VERSION)-$(SHA_SHORT)" \
	--metadata-file metadata.json \
	--push="$(BUILDX_PUSH)" \
	--file Dockerfile.$(BIN) \
	.
ifeq ($(BUILDX_TAG_LATEST),true)
	BUILDX_ARGS += --tag "$(IMAGE):latest"
endif

build-image-setup: $(BUILD_DIRS)
	@echo "Setting up buildx"
	@docker run --rm --privileged tonistiigi/binfmt:latest --install all
	@docker buildx inspect $(BUILDX_NAME) 2>/dev/null || docker buildx create --name $(BUILDX_NAME) --driver docker-container
	@docker buildx use $(BUILDX_NAME)
	@docker buildx ls
	@docker buildx inspect $(BUILDX_NAME)

	@echo "Generating Dockerfile.$(BIN)"
	@cp Dockerfile.tmpl Dockerfile.$(BIN)
	sed -i 's/{{ $$BIN }}/$(BIN)/g' Dockerfile.$(BIN)

	@echo "Running docker buildx"
	@mkdir -p /tmp/.buildx-cache
	@echo "IMAGE: $(IMAGE)"
	@echo "VERSION: $(VERSION)"
	@echo "SHA_SHORT: $(SHA_SHORT)"

build-image: build build-image-setup
	docker buildx build $(BUILDX_ARGS) --platform $(OS)/$(ARCH) --load
	@docker history --no-trunc "$(IMAGE):$(VERSION)"
	@docker inspect "$(IMAGE):$(VERSION)"

buildx-image: all-build build-image-setup
	docker buildx build $(BUILDX_ARGS) --platform $(shell echo $(ALL_PLATFORMS) | tr ' ' ',' )

# Example: make shell CMD="-c 'date > datefile'"
shell: $(BUILD_DIRS)
	@echo "launching a shell in the containerized build environment"
	@docker run \
		-ti \
		--rm \
		-u $$(id -u):$$(id -g) \
		-e GO111MODULE="$(GO111MODULE)" \
		-e GOFLAGS="$(GOFLAGS)" \
		-v $(PWD):$(PWD) \
		-w $(PWD) \
		-v $(PWD)/$(BUILD_GOPATH):/go \
		-v $(PWD)/$(BUILD_GOCACHE):/.cache \
		--env HTTP_PROXY=$(HTTP_PROXY) \
		--env HTTPS_PROXY=$(HTTPS_PROXY) \
		$(BUILD_IMAGE) \
		/bin/sh $(CMD)

# We replace .go and .cache with empty directories in the container
test: $(BUILD_DIRS)
	@docker run \
		-i \
		--rm \
		-u $$(id -u):$$(id -g) \
		-v $(PWD):$(PWD) \
		-w $(PWD) \
		-v $(PWD)/$(BUILD_GOPATH):/go \
		-v $(PWD)/$(BUILD_GOCACHE):/.cache \
		--env HTTP_PROXY=$(HTTP_PROXY) \
		--env HTTPS_PROXY=$(HTTPS_PROXY) \
		$(BUILD_IMAGE) \
		/bin/sh -c " \
			ARCH=$(ARCH) \
			OS=$(OS) \
			VERSION=$(VERSION) \
			GO111MODULE=$(GO111MODULE) \
			GOFLAGS=$(GOFLAGS) \
			COVERAGE_FILE=$(COVERAGE_FILE) \
			./build/test.sh $(SRC_DIRS) \
		"

coverage:
	@$(MAKE) test

checksums: $(BUILD_DIRS) checksums-clean
	@cd $(BUILD_BIN_DIR); for i in $$(ls); do \
		sha1sum "$$i" > "$$i.sha1"; echo "$(BUILD_BIN_DIR)/$$i.sha1";	\
		sha256sum "$$i" > "$$i.sha256"; echo "$(BUILD_BIN_DIR)/$$i.sha256"; \
		sha512sum "$$i" > "$$i.sha512"; echo "$(BUILD_BIN_DIR)/$$i.sha512"; \
	done

# Development docker-compose up. Run build first
DEV_DOCKER_COMPOSE_YML := docker-compose.yml
up: $(DEV_DOCKER_COMPOSE_YML)
	@$(MAKE) build
	@OUTBIN=$(OUTBIN) BIN=$(BIN) UID=$$(id -u) GID=$$(id -g) docker-compose -f $(DEV_DOCKER_COMPOSE_YML) up

# Development docker-compose down
down: $(DEV_DOCKER_COMPOSE_YML)
	@OUTBIN=$(OUTBIN) BIN=$(BIN) UID=$$(id -u) GID=$$(id -g) docker-compose -f $(DEV_DOCKER_COMPOSE_YML) down

# Mounts a ramdisk on ./go/bin
mount-ramdisk:
	@mkdir -p $(BUILD_BIN_DIR)
	@mount | grep $(BUILD_BIN_DIR) && echo "tmpfs already mounted on $(BUILD_BIN_DIR)" || ( sudo mount -t tmpfs -o size=128M tmpfs $(BUILD_BIN_DIR) && mount | grep $(BUILD_BIN_DIR) && echo "tmpfs mounted on $(BUILD_BIN_DIR)" )

# Unmounts a ramdisk on ./go/bin
unmount-ramdisk:
	@mount | grep $(BUILD_BIN_DIR) && sudo umount $(BUILD_BIN_DIR) && echo "unmount $(BUILD_BIN_DIR)" || echo "nothing to unmount on $(BUILD_BIN_DIR)"

clean: bin-clean build-image-clean

bin-clean:
	chmod -R +w $(BUILD_DIRS)
	rm -rf $(BUILD_DIRS)

build-image-clean:
	rm -f Dockerfile
	rm -f metadata.json

checksums-clean:
	rm -f $(BUILD_BIN_DIR)/*.sha1
	rm -f $(BUILD_BIN_DIR)/*.sha256
	rm -f $(BUILD_BIN_DIR)/*.sha512

version:
	@echo $(VERSION)
