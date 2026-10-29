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

COPY --parents libraries/*/go.mod libraries/*/go.sum libraries/*/*/go.mod libraries/*/*/go.sum services/*/go.mod services/*/go.sum services/*/contract/go.mod services/*/contract/go.sum distributions/*/go.mod distributions/*/go.sum ./

WORKDIR /src/distributions/yacypeer

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
        -o /out/yacypeer \
        ./cmd/yacypeer

RUN mkdir -p /out/yacypeer-workdir

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/yacypeer /usr/local/bin/yacypeer
COPY --from=build --chown=nonroot:nonroot /out/yacypeer-workdir /var/lib/yacypeer

ENV YACYPEER_WEB_ADDR=:9090

EXPOSE 8090 8443 9090

WORKDIR /var/lib/yacypeer

USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/yacypeer"]
