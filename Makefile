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
GCR_REGISTRY ?= gcr.io/velero-gcp

# Image name
IMAGE ?= $(REGISTRY)/$(BIN)
GCR_IMAGE ?= $(GCR_REGISTRY)/$(BIN)

# We allow the Dockerfile to be configurable to enable the use of custom Dockerfiles
# that pull base images from different registries.
VELERO_DOCKERFILE ?= Dockerfile
VELERO_DOCKERFILE_WINDOWS ?= Dockerfile-Windows

# Which architecture to build - see $(ALL_ARCH) for options.
# if the 'local' rule is being run, detect the ARCH from 'go env'
# if it wasn't specified by the caller.
local : ARCH ?= $(shell go env GOOS)-$(shell go env GOARCH)
ARCH ?= linux-amd64

VERSION ?= main

TAG_LATEST ?= false

ifeq ($(TAG_LATEST), true)
	IMAGE_TAGS ?= $(IMAGE):$(VERSION) $(IMAGE):latest
	GCR_IMAGE_TAGS ?= $(GCR_IMAGE):$(VERSION) $(GCR_IMAGE):latest
else
	IMAGE_TAGS ?= $(IMAGE):$(VERSION)
	GCR_IMAGE_TAGS ?= $(GCR_IMAGE):$(VERSION)
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

comma=,

CLI_PLATFORMS ?= linux-amd64 linux-arm linux-arm64 darwin-amd64 darwin-arm64 windows-amd64 linux-ppc64le
BUILDX_PUSH ?= false
BUILDX_BUILD_OS ?= linux
BUILDX_BUILD_ARCH ?= amd64
BUILDX_TAG_GCR ?= false
BUILDX_WINDOWS_VERSION ?= ltsc2022

ifneq ($(BUILDX_PUSH), true)
	ALL_OS = linux
	ALL_ARCH.linux = $(word 2, $(subst -, ,$(shell go env GOOS)-$(shell go env GOARCH)))
	BUILDX_OUTPUT_TYPE = docker
else
	ALL_OS = $(subst $(comma), ,$(BUILDX_BUILD_OS))
	ALL_ARCH.linux = $(subst $(comma), ,$(BUILDX_BUILD_ARCH))
	BUILDX_OUTPUT_TYPE = registry
endif

ALL_ARCH.windows = $(if $(filter windows,$(ALL_OS)),amd64,)
ALL_OSVERSIONS.windows = $(if $(filter windows,$(ALL_OS)),$(BUILDX_WINDOWS_VERSION),)
ALL_OS_ARCH.linux =  $(foreach os, $(filter linux,$(ALL_OS)), $(foreach arch, ${ALL_ARCH.linux}, ${os}-$(arch)))
ALL_OS_ARCH.windows = $(foreach os, $(filter windows,$(ALL_OS)), $(foreach arch, $(ALL_ARCH.windows), $(foreach osversion, ${ALL_OSVERSIONS.windows}, ${os}-${osversion}-${arch})))
ALL_OS_ARCH = $(ALL_OS_ARCH.linux)$(ALL_OS_ARCH.windows)

ALL_IMAGE_TAGS = $(IMAGE_TAGS)
ifeq ($(BUILDX_TAG_GCR), true)
	ALL_IMAGE_TAGS += $(GCR_IMAGE_TAGS)
endif

# set git sha and tree state
GIT_SHA = $(shell git rev-parse HEAD)
ifneq ($(shell git status --porcelain 2> /dev/null),)
	GIT_TREE_STATE ?= dirty
else
	GIT_TREE_STATE ?= clean
endif

###
### These variables should not need tweaking.
###

platform_temp = $(subst -, ,$(ARCH))
GOOS = $(word 1, $(platform_temp))
GOARCH = $(word 2, $(platform_temp))
GOPROXY ?= https://proxy.golang.org

local: build-dirs
	GOOS=$(GOOS) \
	GOARCH=$(GOARCH) \
	VERSION=$(VERSION) \
	REGISTRY=$(REGISTRY) \
	PKG=$(PKG) \
	BIN=$(BIN) \
	GIT_SHA=$(GIT_SHA) \
	GIT_TREE_STATE=$(GIT_TREE_STATE) \
	OUTPUT_DIR=$$(pwd)/_output/bin/$(GOOS)/$(GOARCH) \
	./hack/build.sh

# test runs unit tests using 'go test' in the local environment.
test:
	CGO_ENABLED=0 go test -v -coverprofile=coverage.out -timeout 60s ./...

# ci is a convenience target for CI builds.
ci: verify-modules test

container:
ifneq ($(BUILDX_ENABLED), true)
	$(error $(BUILDX_ERROR))
endif
	-docker buildx rm aws-plugin-builder || true
	@docker buildx create --use --name=aws-plugin-builder

	@for osarch in $(ALL_OS_ARCH); do \
		$(MAKE) container-$${osarch}; \
	done

ifeq ($(BUILDX_PUSH), true)
	@for tag in $(ALL_IMAGE_TAGS); do \
		IMAGE_TAG=$${tag} $(MAKE) push-manifest; \
	done
endif

container-linux-%:
	@BUILDX_ARCH=$* $(MAKE) container-linux

container-linux:
	@echo "building container: $(IMAGE):$(VERSION)-linux-$(BUILDX_ARCH)"

	@docker buildx build --pull \
	--output=type=$(BUILDX_OUTPUT_TYPE) \
	--platform="linux/$(BUILDX_ARCH)" \
	$(addprefix -t , $(addsuffix "-linux-$(BUILDX_ARCH)",$(ALL_IMAGE_TAGS))) \
	--build-arg=GOPROXY=$(GOPROXY) \
	--build-arg=PKG=$(PKG) \
	--build-arg=BIN=$(BIN) \
	--build-arg=VERSION=$(VERSION) \
	--build-arg=GIT_SHA=$(GIT_SHA) \
	--build-arg=GIT_TREE_STATE=$(GIT_TREE_STATE) \
	--build-arg=REGISTRY=$(REGISTRY) \
	--provenance=false \
	--sbom=false \
	-f $(VELERO_DOCKERFILE) .
	
	@echo "built container: $(IMAGE):$(VERSION)-linux-$(BUILDX_ARCH)"

container-windows-%:
	@BUILDX_OSVERSION=$(firstword $(subst -, ,$*)) BUILDX_ARCH=$(lastword $(subst -, ,$*)) $(MAKE) container-windows

container-windows:
	@echo "building container: $(IMAGE):$(VERSION)-windows-$(BUILDX_OSVERSION)-$(BUILDX_ARCH)"

	@docker buildx build --pull \
	--output=type=$(BUILDX_OUTPUT_TYPE) \
	--platform="windows/$(BUILDX_ARCH)" \
	$(addprefix -t , $(addsuffix "-windows-$(BUILDX_OSVERSION)-$(BUILDX_ARCH)",$(ALL_IMAGE_TAGS))) \
	--build-arg=GOPROXY=$(GOPROXY) \
	--build-arg=PKG=$(PKG) \
	--build-arg=BIN=$(BIN) \
	--build-arg=VERSION=$(VERSION) \
	--build-arg=OS_VERSION=$(BUILDX_OSVERSION) \
	--build-arg=GIT_SHA=$(GIT_SHA) \
	--build-arg=GIT_TREE_STATE=$(GIT_TREE_STATE) \
	--build-arg=REGISTRY=$(REGISTRY) \
	--provenance=false \
	--sbom=false \
	-f $(VELERO_DOCKERFILE_WINDOWS) .	

	@echo "built container: $(IMAGE):$(VERSION)-windows-$(BUILDX_OSVERSION)-$(BUILDX_ARCH)"

push-manifest:
	@echo "building manifest: $(IMAGE_TAG) for $(foreach osarch, $(ALL_OS_ARCH), $(IMAGE_TAG)-${osarch})"
	@docker manifest create --amend $(IMAGE_TAG) $(foreach osarch, $(ALL_OS_ARCH), $(IMAGE_TAG)-${osarch})

	@set -x; \
	for arch in $(ALL_ARCH.windows); do \
		for osversion in $(ALL_OSVERSIONS.windows); do \
			BASEIMAGE=mcr.microsoft.com/windows/nanoserver:$${osversion}; \
			full_version=`docker manifest inspect $${BASEIMAGE} | jq -r '.manifests[0].platform["os.version"]'`; \
			docker manifest annotate --os windows --arch $${arch} --os-version $${full_version} $(IMAGE_TAG) $(IMAGE_TAG)-windows-$${osversion}-$${arch}; \
		done; \
	done

	@echo "pushing mainifest $(IMAGE_TAG)"
	@docker manifest push --purge $(IMAGE_TAG)

	@echo "pushed mainifest $(IMAGE_TAG):"
	@docker manifest inspect $(IMAGE_TAG)

build-dirs:
	@mkdir -p _output/bin/$(GOOS)/$(GOARCH)

.PHONY: modules
modules:
	go mod tidy

.PHONY: verify-modules
verify-modules: modules
	@if !(git diff --quiet HEAD -- go.sum go.mod); then \
		echo "go module files are out of date, please commit the changes to go.mod and go.sum"; exit 1; \
	fi


changelog:
	hack/release-tools/changelog.sh

# clean removes build artifacts from the local environment.
clean:
	@echo "cleaning"
	rm -rf _output
