# 2. Use go-proxyproto to read forwarded source addresses

Date: 2026-09-02

## Status

Accepted

## Context

The tunnel client gives each forwarded connection to the private ingress behind a PROXY
protocol header. That header holds the only true source address of the request. The ingress
must read the header, refuse a connection that does not send one, and bound the time a slow
connection can hold the header open.

## Decision

Use `github.com/pires/go-proxyproto`. Its `Listener` wraps the ingress listener, applies a
REQUIRE policy, applies a header read timeout, and returns connections whose `RemoteAddr`
is the address the header carries.

## Consequences

`go-proxyproto` becomes a runtime dependency of localhostrunagent, pinned in `go.mod`. It
brings `golang.org/x/net`, which other modules of this repository already use. PROXY
protocol vocabulary stays in the `internal/proxyprotocolingress` package.
