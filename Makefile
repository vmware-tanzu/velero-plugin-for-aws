# Copyright the Velero contributors.
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
BIN ?= velero-plugin-for-aws

# This repo's root import path (under GOPATH).
PKG := github.com/vmware-tanzu/velero-plugin-for-aws

# Where to push the docker image.
REGISTRY ?= velero

# Image name
IMAGE ?= $(REGISTRY)/$(BIN)

# Which architecture to build - see $(ALL_ARCH) for options.
# if the 'local' rule is being run, detect the ARCH from 'go env'
# if it wasn't specified by the caller.
local : ARCH ?= $(shell go env GOOS)-$(shell go env GOARCH)
ARCH ?= linux-amd64

VERSION ?= main

TAG_LATEST ?= false

ifeq ($(TAG_LATEST), true)
    IMAGE_TAGS ?= $(IMAGE):$(VERSION) $(IMAGE):latest
else
    IMAGE_TAGS ?= $(IMAGE):$(VERSION)
endif

ifeq ($(shell docker buildx inspect 2>/dev/null | awk '/Status/ { print $$2 }'), running)
    BUILDX_ENABLED ?= true
else
    BUILDX_ENABLED ?= false
endif

define BUILDX_ERROR
buildx not enabled, refusing to run this recipe
see: https://velero.io/docs/main/build-from-source/#making-images-and-updating-velero for more info
endef

BUILDX_PLATFORMS ?= $(subst -,/,$(ARCH))
BUILDX_OUTPUT_TYPE ?= docker

###
### These variables should not need tweaking.
###

platform_temp = $(subst -, ,$(ARCH))
GOOS = $(word 1, $(platform_temp))
GOARCH = $(word 2, $(platform_temp))
GOPROXY ?= https://proxy.golang.org

# If you want to build all containers, see the 'all-containers' rule.
all-containers:
	@$(MAKE) --no-print-directory container

# local builds the binary using 'go build' in the local environment.
local: build-dirs
	GOOS=$(GOOS) \
	GOARCH=$(GOARCH) \
	PKG=$(PKG) \
	BIN=$(BIN) \
	OUTPUT_DIR=$$(pwd)/_output \
	./hack/build.sh

# test runs unit tests using 'go test' in the local environment.
test:
	CGO_ENABLED=0 go test -v -timeout 60s ./...

# ci is a convenience target for CI builds.
ci: verify-modules test local

# container builds a Docker image containing the binary.
container:
ifneq ($(BUILDX_ENABLED), true)
    $(error $(BUILDX_ERROR))
endif
	@docker buildx build --pull \
    --output=type=$(BUILDX_OUTPUT_TYPE) \
    --platform $(BUILDX_PLATFORMS) \
    $(addprefix -t , $(IMAGE_TAGS)) \
    -f Dockerfile .
	@echo "container: $(IMAGE):$(VERSION)"

# build-dirs creates the necessary directories for a build in the local environment.
build-dirs:
	@mkdir -p _output

.PHONY: modules
modules:
	go mod tidy

.PHONY: verify-modules
verify-modules: modules
	@if !(git diff --quiet HEAD -- go.sum go.mod); then \
		echo "go module files are out of date, please commit the changes to go.mod and go.sum"; exit 1; \
	fi

# clean removes build artifacts from the local environment.
clean:
	@echo "cleaning"
	rm -rf _output
