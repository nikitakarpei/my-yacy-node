# 1. Run the NATS server inside the peer

Status: accepted

## Context

`yacynode`, `corpustext`, and `corpusmarkdown` read page offers from NATS JetStream.
Without a built-in server, an operator of `yacypeer` must also run and connect a NATS
server.

`github.com/nats-io/nats-server/v2` is the NATS server as a Go module, under the Apache 2.0
licence. The repository already pins it for tests.

## Decision

Link `github.com/nats-io/nats-server/v2` into `yacypeer`, and run it with JetStream on the
local host only.

## Consequences

* The peer needs no NATS server outside it.
* Nothing outside the host can connect to the built-in NATS server.
* The binary is larger.
