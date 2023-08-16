# Copyright 2017 The Kubernetes Authors.
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
.PHONY: test build container push clean

PROJECT_DIR=/app
REGISTRY_NAME=docker.io
IMAGE_NAME=majst01/csi-driver-s3
DOCKER_TAG := $(or ${GITHUB_TAG_NAME}, latest)
IMAGE_TAG=$(REGISTRY_NAME)/$(IMAGE_NAME):$(DOCKER_TAG)
TEST_IMAGE_TAG=$(REGISTRY_NAME)/$(IMAGE_NAME):test

SHA := $(shell git rev-parse --short=8 HEAD)
GITVERSION := $(shell git describe --long --all)
BUILDDATE := $(shell date -Iseconds)
VERSION := $(or ${DOCKER_TAG},devel)

build: bin/s3driver

bin/s3driver: pkg/s3/*.go cmd/s3driver/*.go
	CGO_ENABLED=0 GOOS=linux \
	go build -a -trimpath -ldflags "-extldflags '-static' \
                          -X 'github.com/metal-stack/v.Version=$(VERSION)' \
                          -X 'github.com/metal-stack/v.Revision=$(GITVERSION)' \
                          -X 'github.com/metal-stack/v.GitSHA1=$(SHA)' \
                          -X 'github.com/metal-stack/v.BuildDate=$(BUILDDATE)'" \
		-o $@ ./cmd/s3driver
	strip bin/s3driver

test:
	docker build -t $(TEST_IMAGE_TAG) -f test/Dockerfile .
	docker run --rm --privileged -v $(PWD):$(PROJECT_DIR) --device /dev/fuse $(TEST_IMAGE_TAG)
clean:
	go clean -r -x
	-rm -rf bin
	
dockerimage: Dockerfile
	docker build -t $(IMAGE_TAG)  .

dockerpush: dockerimage
	docker push $(IMAGE_TAG)
