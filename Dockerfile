FROM golang:1.17-alpine3.14 as builder

ARG GIT_TOKEN
ARG SDK_TARGET

RUN set -eux; apk add --no-cache ca-certificates build-base;

RUN apk add git jq bash findutils

RUN go env -w GOPRIVATE=github.com/emerishq/*
RUN git config --global url."https://git:${GIT_TOKEN}@github.com".insteadOf "https://github.com"

WORKDIR /app
COPY go.mod go.sum* ./
COPY . .
RUN make clean

# Embedding libwasmvm in all docker images, needed for Terra support
# even if in v42 it's useless.
ADD https://github.com/CosmWasm/wasmvm/releases/download/v0.16.3/libwasmvm_muslc.a /lib/libwasmvm_muslc.a

RUN CGO_ENABLED=1 GOPROXY=direct make setup-${SDK_TARGET}
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=1 GOPROXY=direct make build-${SDK_TARGET}


FROM alpine:latest

RUN apk --no-cache add ca-certificates mailcap && addgroup -S app && adduser -S app -G app

# Add it here too because it's needed at runtime
ADD https://github.com/CosmWasm/wasmvm/releases/download/v0.16.3/libwasmvm_muslc.a /lib/libwasmvm_muslc.a

COPY --from=builder /app/build/sdk_utilities /usr/local/bin/sdk_utilities
USER app
ENTRYPOINT ["/usr/local/bin/sdk_utilities"]
