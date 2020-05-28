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
IMAGE 		?= $(REGISTRY)/velero-plugin-for-aws
VERSION 	?= master

# Which architecture to build.
# if the 'local' rule is being run, detect the GOOS/GOARCH from 'go env'
# if it wasn't specified by the caller.
local: GOOS ?= $(shell go env GOOS)
GOOS ?= linux

local: GOARCH ?= $(shell go env GOARCH)
GOARCH ?= amd64

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
ci: verify-modules test

# container builds a Docker image containing the binary.
.PHONY: container
container:
	docker build -t $(IMAGE):$(VERSION) .

# push pushes the Docker image to its registry.
.PHONY: push
push: container
	@docker push $(IMAGE):$(VERSION)
ifeq ($(TAG_LATEST), true)
	docker tag $(IMAGE):$(VERSION) $(IMAGE):latest
	docker push $(IMAGE):latest
endif

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
