# 24. Serve HTTP and TLS on one peer port

Date: 2026-10-29

## Status

Accepted

## Context

The node served the peer protocol over HTTP on one port and over TLS on a second port. Each
operator had to open or forward two ports, and the node had two listen and two advertise
settings.

## Decision

The node serves HTTP and TLS on the port of `YACY_PEER_ADDR`. `YACY_SECURE_PEER_ADDR` and
`YACY_ADVERTISE_SECURE_PORT` are removed. The seed announces `YACY_ADVERTISE_PORT` as `Port`
and as `PortSSL`.

- The node reads the first byte of each connection. A TLS handshake record (`0x16`) goes to
  TLS, all else goes to HTTP. Both protocols start with the client, so the byte is always
  there.
- A connection that sends no byte in 10 seconds is closed.
- The caller still chooses the scheme. A bound peer is called over TLS only, so one port does
  not open a downgrade.
- `yacynode_peerport_connections_total` counts connections by scheme.

## Consequences

An operator opens one port. A Java peer that prefers HTTPS reaches the node at the same port.

A reverse proxy in front of the port must pass TLS through, as before.
