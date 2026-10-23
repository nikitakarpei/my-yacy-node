# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.27 AS build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

ENV CGO_ENABLED=0 GOFLAGS=-trimpath GOWORK=off

WORKDIR /src

COPY --parents libraries/*/go.mod libraries/*/go.sum libraries/*/*/go.mod libraries/*/*/go.sum services/*/go.mod services/*/go.sum services/*/contract/go.mod services/*/contract/go.sum ./

WORKDIR /src/services/corpusmarkdown

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . /src

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build \
        -ldflags="-s -w -X main.version=${VERSION}" \
        -o /out/corpusmarkdown \
        ./cmd/corpusmarkdown

RUN mkdir -p /out/corpusmarkdown-workdir

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build --chown=nonroot:nonroot /out/corpusmarkdown-workdir /var/lib/corpusmarkdown
COPY --from=build /out/corpusmarkdown /usr/local/bin/corpusmarkdown

WORKDIR /var/lib/corpusmarkdown

USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/corpusmarkdown"]
