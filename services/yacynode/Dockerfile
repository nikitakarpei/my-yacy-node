# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.27 AS build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

ENV CGO_ENABLED=0 GOFLAGS=-trimpath GOWORK=off

WORKDIR /src

RUN apt-get update \
    && apt-get install -y --no-install-recommends xz-utils \
    && rm -rf /var/lib/apt/lists/*

COPY tools/install tools/tools.lock tools/

RUN tools/install zig

COPY --parents libraries/*/go.mod libraries/*/go.sum libraries/*/*/go.mod libraries/*/*/go.sum services/*/go.mod services/*/go.sum services/*/contract/go.mod services/*/contract/go.sum ./

WORKDIR /src/services/yacynode

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . /src

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/.cache/zig \
    set -eu; \
    case "$TARGETARCH" in \
        amd64) zig_target=x86_64-linux-musl;; \
        arm64) zig_target=aarch64-linux-musl;; \
        *) echo "no Zig target for $TARGETARCH" >&2; exit 1;; \
    esac; \
    CGO_ENABLED=1 CC="/src/.toolchain/bin/zig cc -target $zig_target" \
    ZIG_GLOBAL_CACHE_DIR=/root/.cache/zig ZIG_LOCAL_CACHE_DIR=/root/.cache/zig \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build \
        -tags netgo,osusergo \
        -ldflags="-s -w -linkmode external -extldflags -static -X main.version=${VERSION}" \
        -o /out/yacy-rwi-node \
        ./cmd/yacy-rwi-node

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build \
        -ldflags="-s -w" \
        -o /out/yacy-entrypoint \
        ./cmd/yacy-entrypoint

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build \
        -ldflags="-s -w" \
        -o /out/yacy-peer-key \
        ./cmd/yacy-peer-key

RUN mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot

ENV YACY_DATA_DIR=/data \
    YACY_PEER_ADDR=:8090 \
    YACY_SECURE_PEER_ADDR=:8443 \
    YACY_OPS_ADDR=:9090

COPY --from=build --chown=nonroot:nonroot /out/data /data
COPY --from=build /out/yacy-rwi-node /usr/local/bin/yacy-rwi-node
COPY --from=build /out/yacy-entrypoint /usr/local/bin/yacy-entrypoint
COPY --from=build /out/yacy-peer-key /usr/local/bin/yacy-peer-key

EXPOSE 8090 8443 9090

VOLUME ["/data"]

USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/yacy-entrypoint"]
CMD ["/usr/local/bin/yacy-rwi-node"]
