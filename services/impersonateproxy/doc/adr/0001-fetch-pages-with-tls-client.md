# 1. Fetch pages with tls-client

Date: 2026-10-10

## Status

Accepted

## Context

Some origins refuse or change the pages they send to clients that do not look
like a browser. They compare the TLS ClientHello, the HTTP/2 SETTINGS, window
and pseudo-header order, and the header order with those of real browsers. The
Go standard library cannot send the ClientHello or the HTTP/2 frames of Chrome.

## Decision

- `go.mod` pins `github.com/bogdanfinn/tls-client` and its HTTP library
  `github.com/bogdanfinn/fhttp`.
- The proxy fetches each page with a Chrome profile of tls-client, through the
  egress proxy in a `CONNECT` tunnel.
- The proxy decodes response bodies with the decoders of fhttp.
- The licence of tls-client is BSD-4-Clause. The owner approved it.

## Consequences

tls-client, fhttp and their dependencies become runtime dependencies. Only the
`internal/pagefetchers/tlsclient` package uses them. A new Chrome version needs
a tls-client release with its profile.
