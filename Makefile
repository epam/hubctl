# Copyright (c) 2022 EPAM Systems, Inc.
#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.

.DEFAULT_GOAL := build

OS := $(shell go env GOOS)

export GOBIN := $(abspath .)/bin/$(OS)
export PATH  := $(GOBIN):$(PATH)

ifeq (,$(METRICS_API_SECRET))
$(info METRICS_API_SECRET is not set - usage metrics won't be submitted to SuperHub API; \
see https://github.com/agilestacks/documentation/wiki/Hub-CLI-Usage-Metrics)
endif

ifeq (,$(DD_CLIENT_API_KEY))
$(info DD_CLIENT_API_KEY is not set - usage metrics won't be submitted to Datadog; \
see https://github.com/agilestacks/documentation/wiki/Hub-CLI-Usage-Metrics)
endif

REF      ?= $(shell git rev-parse --abbrev-ref HEAD)
COMMIT   ?= $(shell git rev-parse HEAD | cut -c-7)
BUILD_AT ?= $(shell date +"%Y.%m.%d %H:%M %Z")

install: bin/$(OS)/gocloc bin/$(OS)/staticcheck
bin/$(OS)/gocloc:
	go install github.com/hhatto/gocloc/cmd/gocloc@latest
bin/$(OS)/staticcheck:
	go install honnef.co/go/tools/cmd/staticcheck@2022.1.2

cel:
	go get github.com/agilestacks/hub/cmd/cel
.PHONY: cel

build:
	go build \
		-o bin/$(OS)/hub \
		-ldflags="-s -w \
		-X 'github.com/agilestacks/hub/cmd/hub/util.ref=$(REF)' \
		-X 'github.com/agilestacks/hub/cmd/hub/util.commit=$(COMMIT)' \
		-X 'github.com/agilestacks/hub/cmd/hub/util.buildAt=$(BUILD_AT)' \
		-X 'github.com/agilestacks/hub/cmd/hub/metrics.MetricsServiceKey=$(METRICS_API_SECRET)' \
		-X 'github.com/agilestacks/hub/cmd/hub/metrics.DDKey=$(DD_CLIENT_API_KEY)'" \
		github.com/agilestacks/hub/cmd/hub
.PHONY: get

fmt:
	go fmt github.com/agilestacks/hub/...
.PHONY: fmt

vet:
	go vet -composites=false github.com/agilestacks/hub/...
.PHONY: vet

loc: bin/$(OS)/gocloc
	@$(GOBIN)/gocloc cmd/hub --not-match-d='cmd/hub/bindata'
.PHONY: loc

staticcheck: bin/$(OS)/staticcheck
	@$(GOBIN)/staticcheck github.com/agilestacks/hub/...
.PHONY: staticcheck

test:
	go test -timeout 30s github.com/agilestacks/hub/cmd/hub/...
.PHONY: test

clean:
	@rm -f hub cel bin/hub bin/cel
	@rm -rf bin/darwin* bin/linux* bin/windows*
.PHONY: clean
