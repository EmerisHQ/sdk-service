OBJS = $(shell find cmd -mindepth 1 -type d -execdir printf '%s\n' {} +)

TARGETS = sdk_targets.json

SHELL := /usr/bin/env bash

SETUP_VERSIONS = $(shell jq -r 'map(.version |= "setup-\(.)")[].version' ${TARGETS})
BUILD_VERSIONS = $(shell jq -r 'map(.version |= "build-\(.)")[].version' ${TARGETS})

BASEPKG = github.com/allinbits/sdk-service
.PHONY: $(OBJS) goagenerate clean $(SETUP_VERSIONS) $(BUILD_VERSIONS)


goagenerate:
	rm -rf cmd gen
	goa example github.com/allinbits/sdk-service-meta
	find . -type f -name '*.go' -exec sed -i "s|github.com/allinbits/sdk-service/gen|github.com/allinbits/sdk-service-meta/gen|g" {} +

$(BUILD_VERSIONS):
	go build -o build/sdk_utilities -v \
	 -tags $(shell echo $@ | sed 's/build-/sdk_/g') \
	 -ldflags "-X main.SupportedSDKVersion=$(shell echo $@ | sed 's/build-//g')" \
	 ${BASEPKG}/cmd/sdk_utilities
	
	go build -o build/sdk_utilities-cli -v \
	 -tags $(shell echo $@ | sed 's/build-/sdk_/g') \
	 -ldflags "-X main.SupportedSDKVersion=$(shell echo $@ | sed 's/build-//g')" \
	 ${BASEPKG}/cmd/sdk_utilities-cli
clean:
	rm -rf build
	rm go.mod go.sum | true
	cp mods/go.mod.bare ./go.mod

docker:
	docker build -t emeris/sdk-service --build-arg GIT_TOKEN=${GITHUB_TOKEN} -f Dockerfile .

$(SETUP_VERSIONS):
	if [ -f ".selected_sdk_version" ]; then \
		echo "Clearing old SDK imports"; \
		./contrib/remove-old-imports.sh $(shell cat .selected_sdk_version) ${TARGETS}; \
	fi

	echo $(shell echo $@ | sed 's/setup-//g') > .selected_sdk_version
	
	cp mods/go.mod.$(shell echo $@ | sed 's/setup-//g') ./go.mod
	cp mods/go.sum.$(shell echo $@ | sed 's/setup-//g') ./go.sum

	#go get -tags $(shell echo $@ | sed 's/setup-/sdk_/g') | true
	# ./contrib/set-replaces.sh $(shell echo $@ | sed 's/setup-//g') ${TARGETS}
	# ./contrib/set-imports.sh $(shell echo $@ | sed 's/setup-//g') ${TARGETS}

available-go-tags:
	@echo Available Go \`//go:build\' tags:
	@jq -r 'map(.version |= "\t - sdk_\(.)")[].version' ${TARGETS}

selected-sdk-version:
	@cat .selected_sdk_version

clean-gomod:
	@for i in $(shell jq -r 'map(.version |= "\(.)")[].version' ${TARGETS}) ; do \
		echo "Clearing SDK $$i imports" ; \
		./contrib/remove-old-imports.sh $$i ${TARGETS}; \
	done
versions-json:
	@jq -r -c "map( { "version": .version } )" ${TARGETS}