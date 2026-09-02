# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.27 AS build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

ENV CGO_ENABLED=0 GOFLAGS=-trimpath GOWORK=off

WORKDIR /src

COPY --parents libraries/*/go.mod libraries/*/go.sum libraries/*/*/go.mod libraries/*/*/go.sum services/*/go.mod services/*/go.sum services/*/contract/go.mod services/*/contract/go.sum ./

WORKDIR /src/services/yacynode

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . /src

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build \
        -ldflags="-s -w -X main.version=${VERSION}" \
        -o /out/yacy-rwi-node \
        ./cmd/yacy-rwi-node

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build \
        -ldflags="-s -w" \
        -o /out/yacy-entrypoint \
        ./cmd/yacy-entrypoint

RUN mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot

ENV YACY_DATA_DIR=/data \
    YACY_PEER_ADDR=:8090 \
    YACY_OPS_ADDR=:9090

COPY --from=build --chown=nonroot:nonroot /out/data /data
COPY --from=build /out/yacy-rwi-node /usr/local/bin/yacy-rwi-node
COPY --from=build /out/yacy-entrypoint /usr/local/bin/yacy-entrypoint

EXPOSE 8090 9090

VOLUME ["/data"]

USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/yacy-entrypoint"]
CMD ["/usr/local/bin/yacy-rwi-node"]
