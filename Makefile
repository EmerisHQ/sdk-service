OBJS = $(shell find cmd -mindepth 1 -type d -execdir printf '%s\n' {} +)

.DEFAULT_GOAL = all

MAJ_SDK_VERSION = 42
BASEPKG = "github.com/allinbits/sdk-service-v${MAJ_SDK_VERSION}"
.PHONY: $(OBJS) goagenerate clean

goagenerate:
	rm -rf cmd gen
	goa example github.com/allinbits/sdk-service-meta
	find . -type f -name '*.go' -exec sed -i "s|github.com/allinbits/sdk-service-v${MAJ_SDK_VERSION}/gen|github.com/allinbits/sdk-service-meta/gen|g" {} +

$(OBJS): 
	go build -o build/$@ ${BASEPKG}/cmd/$@
	
all: $(OBJS)

clean:
	rm -rf build

docker:
	docker build -t emeris/sdk-service-v42 --build-arg GIT_TOKEN=${GITHUB_TOKEN} -f Dockerfile .