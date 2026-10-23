# 1. Run the NATS server inside the peer

Status: accepted

## Context

`yacynode`, `corpustext`, and `corpusmarkdown` read page offers from NATS JetStream. An
operator who runs `yacypeer` must otherwise run and connect a NATS server too.

`github.com/nats-io/nats-server/v2` is the NATS server as a Go module, under the Apache 2.0
licence. The repository already pins it for tests.

## Decision

Link `github.com/nats-io/nats-server/v2` into `yacypeer`. The binary starts it with
JetStream, on the loopback interface and on a port the system picks, and keeps its store
in the data directory.

## Consequences

* The peer runs with no NATS server outside it.
* Nothing outside the process can connect to the built-in NATS server.
* The binary grows by the NATS server and its dependencies.
* The NATS server version follows the version the repository pins.
