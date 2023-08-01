# Copyright (c) 2022 EPAM Systems, Inc.
#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.

.DEFAULT_GOAL := build

OS := $(shell go env GOOS)

export GOBIN := $(abspath .)/bin/$(OS)
export PATH  := $(GOBIN):$(PATH)

REF      ?= $(shell git rev-parse --abbrev-ref HEAD)
COMMIT   ?= $(shell git rev-parse HEAD | cut -c-7)
BUILD_AT ?= $(shell date +"%Y.%m.%d %H:%M %Z")

install: bin/$(OS)/gocloc bin/$(OS)/staticcheck bin/$(OS)/gotests
bin/$(OS)/gocloc:
	go install github.com/hhatto/gocloc/cmd/gocloc@latest
bin/$(OS)/staticcheck:
	go install honnef.co/go/tools/cmd/staticcheck@2022.1.2
bin/$(OS)/gotests:
	$ go get -u github.com/cweill/gotests/...

cel:
	go get github.com/epam/hubctl/cmd/cel
.PHONY: cel

build:
	go build \
		-o bin/$(OS)/hubctl \
		-ldflags="-s -w \
		-X 'github.com/epam/hubctl/cmd/hub/util.ref=$(REF)' \
		-X 'github.com/epam/hubctl/cmd/hub/util.commit=$(COMMIT)' \
		-X 'github.com/epam/hubctl/cmd/hub/util.buildAt=$(BUILD_AT)'" \
		github.com/epam/hubctl/cmd/hub
.PHONY: build

build-with-api:
	go build \
		-o "bin/$(OS)/hubctl" \
		-tags="api" \
		-ldflags="-s -w \
		-X 'github.com/epam/hubctl/cmd/hub/util.ref=$(REF)' \
		-X 'github.com/epam/hubctl/cmd/hub/util.commit=$(COMMIT)' \
		-X 'github.com/epam/hubctl/cmd/hub/util.buildAt=$(BUILD_AT)' \
		-X 'github.com/epam/hubctl/cmd/hub/metrics.MetricsServiceKey=$(METRICS_API_SECRET)' \
		-X 'github.com/epam/hubctl/cmd/hub/metrics.DDKey=$(DD_CLIENT_API_KEY)'" \
		github.com/epam/hubctl/cmd/hub
.PHONY: build-with-api

fmt:
	go fmt github.com/epam/hubctl/...
.PHONY: fmt

vet:
	go vet -composites=false github.com/epam/hubctl/...
.PHONY: vet

loc: bin/$(OS)/gocloc
	@$(GOBIN)/gocloc cmd/hub --not-match-d='cmd/hub/bindata'
.PHONY: loc

staticcheck: bin/$(OS)/staticcheck
	@$(GOBIN)/staticcheck github.com/epam/hubctl/...
.PHONY: staticcheck

test:
	go test -timeout 30s github.com/epam/hubctl/cmd/hub/...
.PHONY: test

clean:
	@rm -f hub cel bin/hub bin/cel
	@rm -rf bin/darwin* bin/linux* bin/windows*
.PHONY: clean
