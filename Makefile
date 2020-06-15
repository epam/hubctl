.DEFAULT_GOAL := get

OS := $(shell uname -s | tr A-Z a-z)

export GOBIN := $(abspath .)/bin/$(OS)
export PATH  := $(GOBIN):$(PATH)

export AWS_PROFILE ?=
S3_BUCKET          ?= controlplane.agilestacks.io
S3_DISTRIBUTION    ?= s3://$(S3_BUCKET)/dist/hub-cli

aws := aws

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

compile: bin/$(OS)/gox version
	go mod download
	nice $(GOBIN)/gox -rebuild -tags git \
		-osarch="darwin/amd64 linux/amd64 windows/amd64" \
		-output=bin/{{.OS}}/hub \
		github.com/agilestacks/hub/cmd/hub
.PHONY: compile

distribute: compile
	$(aws) s3 cp bin/darwin/hub      $(S3_DISTRIBUTION)/hub.darwin_amd64
	$(aws) s3 cp bin/linux/hub       $(S3_DISTRIBUTION)/hub.linux_amd64
	$(aws) s3 cp bin/windows/hub.exe $(S3_DISTRIBUTION)/hub.windows_amd64.exe
.PHONY: distribute

undistribute:
	-$(aws) s3 rm $(S3_DISTRIBUTION)/hub.darwin_amd64
	-$(aws) s3 rm $(S3_DISTRIBUTION)/hub.linux_amd64
	-$(aws) s3 rm $(S3_DISTRIBUTION)/hub.windows_amd64.exe
.PHONY: undistribute

cel:
	go get github.com/agilestacks/hub/cmd/cel
.PHONY: cel

get: version
	go get -tags git github.com/agilestacks/hub/cmd/hub
.PHONY: get

bindata: bin/$(OS)/go-bindata
	$(GOBIN)/go-bindata -o cmd/hub/bindata/bindata.go -pkg bindata \
		meta/hub-well-known-parameters.yaml \
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
	@rm -rf bin/darwin bin/linux bin/windows
.PHONY: clean
