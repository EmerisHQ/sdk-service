FROM golang:1.17 as builder

ARG GIT_TOKEN
ARG SDK_TARGET

RUN go env -w GOPRIVATE=github.com/allinbits/*
RUN git config --global url."https://git:${GIT_TOKEN}@github.com".insteadOf "https://github.com"

RUN apt update -y && apt install jq -y

WORKDIR /app
COPY go.mod go.sum* ./
COPY . .
RUN make clean
RUN CGO_ENABLED=0 GOPROXY=direct make setup-${SDK_TARGET}
RUN CGO_ENABLED=0 GOPROXY=direct make build-${SDK_TARGET}

FROM alpine:latest

RUN apk --no-cache add ca-certificates mailcap && addgroup -S app && adduser -S app -G app
COPY --from=builder /app/build/sdk_utilities /usr/local/bin/sdk_utilities
USER app
ENTRYPOINT ["/usr/local/bin/sdk_utilities"]
