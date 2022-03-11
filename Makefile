.DEFAULT_GOAL := get

OS := $(shell uname -s | tr A-Z a-z)

export GOBIN := $(abspath .)/bin/$(OS)
export PATH  := $(GOBIN):$(PATH)

export AWS_PROFILE ?=
S3_BUCKET          ?= controlplane.agilestacks.io
S3_DISTRIBUTION    ?= s3://$(S3_BUCKET)/dist/hub-cli

ifeq (,$(METRICS_API_SECRET))
$(warning METRICS_API_SECRET is not set - usage metrics won't be submitted to SuperHub API; \
see https://github.com/agilestacks/documentation/wiki/Hub-CLI-Usage-Metrics)
endif

ifeq (,$(DD_CLIENT_API_KEY))
$(info DD_CLIENT_API_KEY is not set - usage metrics won't be submitted to Datadog; \
see https://github.com/agilestacks/documentation/wiki/Hub-CLI-Usage-Metrics)
endif

IMAGE         ?= agilestacks/hub
IMAGE_VERSION ?= $(shell git rev-parse HEAD | cut -c-7)
IMAGE_TAG     ?= latest
USER_FULLNAME ?= $(shell id -un)
REGISTRY_PASS ?= ~/.docker/agilestacks.txt

aws    := aws
docker := docker

install: bin/$(OS)/gox bin/$(OS)/go-bindata
bin/$(OS)/gox:
	go get -u github.com/mitchellh/gox
bin/$(OS)/go-bindata:
	go get -u github.com/tmthrgd/go-bindata/...
bin/$(OS)/gocloc:
	go get -u github.com/hhatto/gocloc/cmd/gocloc

version:
	@sed -e s/'\$$version'/"git $(shell git rev-parse HEAD | cut -c-7) built on $(shell date +"%Y.%m.%d %H:%M %Z")"/ < \
		cmd/hub/util/version.go.template > cmd/hub/util/version.go
.PHONY: version

metrics-keys:
	@cat cmd/hub/metrics/keys.go.template | \
		sed -e s/'\$$metricskey'/$$METRICS_API_SECRET/ | \
		sed -e s/'\$$ddkey'/$$DD_CLIENT_API_KEY/ > cmd/hub/metrics/keys.go
.PHONY: metrics-keys

compile: bin/$(OS)/gox version metrics-keys
	go mod download
	nice $(GOBIN)/gox -parallel=4 -rebuild -tags "git metrics"\
		-osarch="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64" \
		-output=bin/hub.{{.OS}}_{{.Arch}} \
		github.com/agilestacks/hub/cmd/hub
.PHONY: compile

distribute: compile
	$(aws) s3 cp --no-progress bin/hub.linux_amd64       $(S3_DISTRIBUTION)/
	$(aws) s3 cp --no-progress bin/hub.linux_arm64       $(S3_DISTRIBUTION)/
	$(aws) s3 cp --no-progress bin/hub.darwin_amd64      $(S3_DISTRIBUTION)/
	$(aws) s3 cp --no-progress bin/hub.darwin_arm64      $(S3_DISTRIBUTION)/
	$(aws) s3 cp --no-progress bin/hub.windows_amd64.exe $(S3_DISTRIBUTION)/
.PHONY: distribute

undistribute:
	-$(aws) s3 rm $(S3_DISTRIBUTION)/hub.linux_amd64
	-$(aws) s3 rm $(S3_DISTRIBUTION)/hub.linux_arm64
	-$(aws) s3 rm $(S3_DISTRIBUTION)/hub.darwin_amd64
	-$(aws) s3 rm $(S3_DISTRIBUTION)/hub.darwin_arm64
	-$(aws) s3 rm $(S3_DISTRIBUTION)/hub.windows_amd64.exe
.PHONY: undistribute

static: GO_FLAGS=-ldflags "-linkmode external -extldflags -static"
static: get

docker: static
	$(docker) build \
		--build-arg="FULLNAME=$(USER_FULLNAME)" \
		--tag $(IMAGE):$(IMAGE_VERSION) \
		--tag $(IMAGE):$(IMAGE_TAG) .
.PHONY: docker

push: login push-version push-tag
push-version:
	$(docker) push $(IMAGE):$(IMAGE_VERSION)
.PHONY: push-version
push-tag:
	$(docker) push $(IMAGE):$(IMAGE_TAG)
.PHONY: push-tag
login:
	@touch $(REGISTRY_PASS)
	@echo "Please put Docker Hub password into $(REGISTRY_PASS)"
	cat $(REGISTRY_PASS) | docker login --username agilestacks --password-stdin
.PHONY: login

cel:
	go get github.com/agilestacks/hub/cmd/cel
.PHONY: cel

get: version metrics-keys
	go get -tags "git metrics" $(GO_FLAGS) github.com/agilestacks/hub/cmd/hub
.PHONY: get

bindata: bin/$(OS)/go-bindata
	$(GOBIN)/go-bindata -o cmd/hub/bindata/bindata.go -pkg bindata \
		meta/hub-well-known-parameters.yaml \
		meta/manifest.schema.json \
		cmd/hub/api/requests/*.template \
		cmd/hub/initialize/hub.yaml.template \
		cmd/hub/initialize/hub-component.yaml.template
.PHONY: bindata

fmt:
	go fmt github.com/agilestacks/hub/...
.PHONY: fmt

vet:
	go vet -composites=false github.com/agilestacks/hub/...
.PHONY: vet

loc: bin/$(OS)/gocloc
	@$(GOBIN)/gocloc cmd/hub --not-match-d='cmd/hub/bindata'
.PHONY: loc

clean:
	@rm -f hub cel bin/hub bin/cel
	@rm -rf bin/darwin* bin/linux* bin/windows*
.PHONY: clean
