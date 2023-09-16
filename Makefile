# Copyright (c) 2023 EPAM Systems, Inc.
#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at https://www.mozilla.org/en-US/MPL/2.0/.

.DEFAULT_GOAL := build

OS := $(shell go env GOOS)

export GOBIN := $(abspath .)/bin/$(OS)
export PATH  := $(GOBIN):$(PATH)

REF      ?= $(shell git rev-parse --abbrev-ref HEAD)
COMMIT   ?= $(shell git rev-parse HEAD | cut -c-7)
BUILD_AT ?= $(shell date +"%Y.%m.%d %H:%M %Z")

install: bin/$(OS)/staticcheck bin/$(OS)/cel
bin/$(OS)/staticcheck:
	go install honnef.co/go/tools/cmd/staticcheck@2023.1.3

cel:
	go install github.com/epam/hubctl/cmd/cel@latest
.PHONY: cel

deps:
	go mod download
.PHONY: deps

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

staticcheck: bin/$(OS)/staticcheck
	@$(GOBIN)/staticcheck github.com/epam/hubctl/...
.PHONY: staticcheck

test:
	go test -race -timeout 60s ./cmd/hub/...
.PHONY: test

clean:
	@go clean -modcache -cache -testcache
	@rm -rf bin/$(OS)*
.PHONY: clean
