OBJS := $(shell find cmd -mindepth 1 -type d -execdir printf '%s\n' {} +)

TARGETS := sdk_targets.json

SHELL := /usr/bin/env bash

SETUP_VERSIONS := $(shell jq -r '.versions|map("setup-\(.)")[]'  ${TARGETS})
BUILD_VERSIONS := $(shell jq -r '.versions|map("build-\(.)")[]' ${TARGETS})
STORE_MOD_VERSIONS := $(shell jq -r '.versions|map("store-mod-\(.)")[]' ${TARGETS})
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')

BASEPKG := github.com/emerishq/sdk-service
.PHONY: $(OBJS) goagenerate clean $(SETUP_VERSIONS) $(BUILD_VERSIONS)


goagenerate:
	rm -rf cmd gen
	goa example github.com/emerishq/sdk-service-meta
	find . -type f -name '*.go' -exec sed -i "s|github.com/emerishq/sdk-service/gen|github.com/emerishq/sdk-service-meta/gen|g" {} +

$(BUILD_VERSIONS):
	go build -o build/sdk_utilities -v \
	 -tags $(shell echo $@ | sed -e 's/build-/sdk_/g' -e 's/-/_/g'),muslc \
	 -ldflags "-X main.Version=${BRANCH}-${COMMIT} -X main.SupportedSDKVersion=$(shell echo $@ | sed -e 's/build-//g' -e 's/-/_/g')" \
	 ${BASEPKG}/cmd/sdk_utilities
	
	go build -o build/sdk_utilities-cli -v \
	 -tags $(shell echo $@ | sed -e 's/build-/sdk_/g' -e 's/-/_/g'),muslc \
	 -ldflags "-X main.SupportedSDKVersion=$(shell echo $@ | sed -e 's/build-//g' -e 's/-/_/g')" \
	 ${BASEPKG}/cmd/sdk_utilities-cli
clean:
	rm -rf build
	rm go.mod go.sum | true
	cp mods/go.mod.bare ./go.mod

docker:
	docker build -t emeris/sdk-service --build-arg GIT_TOKEN=${GITHUB_TOKEN} -f Dockerfile .

$(SETUP_VERSIONS):
	cp mods/go.mod.$(shell echo $@ | sed 's/setup-//g') ./go.mod
	cp mods/go.sum.$(shell echo $@ | sed 's/setup-//g') ./go.sum

available-go-tags:
	@echo "Available Go \`//go:build\' tags":
	@jq -r '.versions|map("sdk_\(.)")[]' ${TARGETS}

versions-json:
	@jq -r -c "map( { "versions": .[] } )" ${TARGETS}

$(STORE_MOD_VERSIONS):
	cp ./go.mod mods/go.mod.$(shell echo $@ | sed 's/store-mod-//g')
	cp ./go.sum mods/go.sum.$(shell echo $@ | sed 's/store-mod-//g')
