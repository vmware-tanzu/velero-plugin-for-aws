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

ORG = vmware-tanzu
PROJECT = velero-plugin-aws
PKG := github.com/$(ORG)/$(PROJECT)
BIN := $(PROJECT)

REGISTRY ?= carlisia
IMAGE ?= $(REGISTRY)/plugin-aws
BUILD_IMAGE ?= golang:1.12-stretch

# Which architecture to build - see $(ALL_ARCH) for options.
# if the 'local' rule is being run, detect the ARCH from 'go env'
# if it wasn't specified by the caller.
local : ARCH ?= $(shell go env GOOS)-$(shell go env GOARCH)
ARCH ?= linux-amd64

VERSION ?= master

platform_temp = $(subst -, ,$(ARCH))
GOOS = $(word 1, $(platform_temp))
GOARCH = $(word 2, $(platform_temp))

all: $(addprefix build-, $(BIN))

build-%:
	$(MAKE) --no-print-directory BIN=$* build

local: build-dirs
	GOOS=$(GOOS) \
	GOARCH=$(GOARCH) \
	PKG=$(PKG) \
	BIN=$(BIN) \
	OUTPUT_DIR=$$(pwd)/_output/bin/$(GOOS)/$(GOARCH) \
	./hack/build.sh

build: _output/bin/$(GOOS)/$(GOARCH)/$(BIN) ./$(PROJECT)

_output/bin/$(GOOS)/$(GOARCH)/$(BIN): build-dirs
	@echo "building: $@"
	$(MAKE) shell CMD="-c '\
		GOOS=$(GOOS) \
		GOARCH=$(GOARCH) \
		PKG=$(PKG) \
		BIN=$(BIN) \
		OUTPUT_DIR=/output/$(GOOS)/$(GOARCH) \
		./hack/build.sh'"

TTY := $(shell tty -s && echo "-t")

shell: build-dirs
	@echo "running docker: $@"
	@docker run \
		-e GOFLAGS \
		-i $(TTY) \
		--rm \
		-u $$(id -u):$$(id -g) \
		-v $$(pwd)/.go/pkg:/go/pkg \
		-v $$(pwd)/.go/src:/go/src \
		-v $$(pwd)/.go/std:/go/std \
		-v $$(pwd):/go/src/$(PKG) \
		-v $$(pwd)/.go/std/$(GOOS)_$(GOARCH):/usr/local/go/pkg/$(GOOS)_$(GOARCH)_static \
		-v "$$(pwd)/.go/go-build:/.cache/go-build:delegated" \
		-e CGO_ENABLED=0 \
		-w /go/src/$(PKG) \
		$(BUILD_IMAGE) \
		go build -installsuffix "static" -i -v -o _output/bin/$(GOOS)/$(GOARCH)/$(BIN) ./$(BIN)

build-dirs:
	@mkdir -p _output/bin/$(GOOS)/$(GOARCH)
	@mkdir -p .go/src/$(PKG) .go/pkg .go/bin .go/std/$(GOOS)/$(GOARCH) .go/go-build

container: all
	cp Dockerfile _output/bin/$(GOOS)/$(GOARCH)/Dockerfile
	docker build -t $(IMAGE):$(VERSION) -f _output/bin/$(GOOS)/$(GOARCH)/Dockerfile _output/bin/$(GOOS)/$(GOARCH)
	
push:
	@docker push $(IMAGE):$(VERSION)

container-name:
	@echo "container: $(IMAGE):$(VERSION)"

all-ci: $(addprefix ci-, $(BIN))

ci-%:
	$(MAKE) --no-print-directory BIN=$* ci

ci:
	mkdir -p _output
	CGO_ENABLED=0 go build -v -o _output/bin/$(GOOS)/$(GOARCH)/$(BIN) ./$(BIN)

clean:
	@echo "cleaning"
	rm -rf .go _output