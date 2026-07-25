# yacy-node

A lightweight, composable reimplementation of a YaCy peer as small Go services.
Each service runs and scales on its own; wire together only the ones a
deployment needs. Services interoperate with the wider YaCy network over DHT.

This is not a port of the full Java YaCy peer. Scope follows the author's own
needs, not parity with the reference implementation — except where a service
must speak an existing protocol or API, where it stays compliant.

## Services

Each directory under `services/` is one deployable service. Its
`doc/specification.md` covers its design; its `doc/configuration.md` covers
its operator-facing settings.

## Start here

`examples/` holds runnable compose stacks that wire services together —
start there to bring the project up.
