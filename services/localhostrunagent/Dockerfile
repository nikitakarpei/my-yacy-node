# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.27 AS build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

ENV CGO_ENABLED=0 GOFLAGS=-trimpath GOWORK=off

WORKDIR /src

COPY --parents libraries/*/go.mod libraries/*/go.sum libraries/*/*/go.mod libraries/*/*/go.sum services/*/go.mod services/*/go.sum services/*/contract/go.mod services/*/contract/go.sum ./

WORKDIR /src/services/localhostrunagent

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . /src

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build \
        -ldflags="-s -w -X main.version=${VERSION}" \
        -o /out/localhostrunagent \
        ./cmd/localhostrunagent

FROM docker.io/library/alpine:3.21

RUN apk add --no-cache openssh-client \
    && adduser -D -H -u 65532 localhostrunagent \
    && mkdir -p /state /run/yacy \
    && chown localhostrunagent:localhostrunagent /state /run/yacy

COPY --from=build /out/localhostrunagent /usr/local/bin/localhostrunagent

USER localhostrunagent

ENTRYPOINT ["/usr/local/bin/localhostrunagent"]
