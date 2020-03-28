# Copyright 2017, 2019 the Velero contributors.
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

PKG := github.com/vmware-tanzu/velero-plugin-for-aws
BIN := velero-plugin-for-aws

REGISTRY 	?= velero
VERSION 	?= master

CONTAINER_PLATFORMS ?= amd64 arm arm64 # ppc64le

# Which architecture to build.
# if the 'local' rule is being run, detect the GOOS/GOARCH from 'go env'
# if it wasn't specified by the caller.
local: GOOS ?= $(shell go env GOOS)
GOOS ?= linux

local: GOARCH ?= $(shell go env GOARCH)
GOARCH ?= amd64

# Set default base image dynamically for each arch
ifeq ($(GOARCH),amd64)
		DOCKERFILE ?= Dockerfile
endif
ifeq ($(GOARCH),arm)
		DOCKERFILE ?= Dockerfile-arm
endif
ifeq ($(GOARCH),arm64)
		DOCKERFILE ?= Dockerfile-arm64
endif


MULTIARCH_IMAGE = $(REGISTRY)/$(BIN)
IMAGE ?= $(REGISTRY)/$(BIN)-$(GOARCH)

# If you want to build all containers, see the 'all-containers' rule.
# If you want to build AND push all containers, see the 'all-push' rule.

container-%:
	@$(MAKE) --no-print-directory GOARCH=$* container

push-%:
	@$(MAKE) --no-print-directory GOARCH=$* push

all-containers: $(addprefix container-, $(CONTAINER_PLATFORMS))

all-push: $(addprefix push-, $(CONTAINER_PLATFORMS))

all-manifests:
	@$(MAKE) manifest

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
ci: test


# container builds a Docker image containing the binary.
container:
	docker build -t $(IMAGE):$(VERSION) -f $(DOCKERFILE) .
	@echo "container: $(IMAGE):$(VERSION)"


# push pushes the Docker image to its registry.
push:
	@docker push $(IMAGE):$(VERSION)
	@echo "pushed: $(IMAGE):$(VERSION)"
ifeq ($(TAG_LATEST), true)
	docker tag $(IMAGE):$(VERSION) $(IMAGE):latest
	docker push $(IMAGE):latest
	@echo "pushed: $(IMAGE):latest"
endif


manifest: 
	DOCKER_CLI_EXPERIMENTAL=enabled docker manifest create $(MULTIARCH_IMAGE):$(VERSION) \
		$(foreach arch, $(CONTAINER_PLATFORMS), $(MULTIARCH_IMAGE)-$(arch):$(VERSION))
	@DOCKER_CLI_EXPERIMENTAL=enabled docker manifest push --purge $(MULTIARCH_IMAGE):$(VERSION)
	@echo "pushed: $(MULTIARCH_IMAGE):$(VERSION)"
ifeq ($(TAG_LATEST), true)
	@DOCKER_CLI_EXPERIMENTAL=enabled docker manifest create $(MULTIARCH_IMAGE):latest \
		$(foreach arch, $(CONTAINER_PLATFORMS), $(MULTIARCH_IMAGE)-$(arch):latest)
	@DOCKER_CLI_EXPERIMENTAL=enabled docker manifest push --purge $(MULTIARCH_IMAGE):latest
	@echo "pushed: $(MULTIARCH_IMAGE):latest)"
endif


# build-dirs creates the necessary directories for a build in the local environment.
build-dirs:
	@mkdir -p _output

# clean removes build artifacts from the local environment.
clean:
	@echo "cleaning"
	rm -rf _output
