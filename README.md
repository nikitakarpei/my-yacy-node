# yacy-node

A lightweight, composable reimplementation of a YaCy peer as small Go services.
Each service runs and scales on its own; wire together only the ones a
deployment needs. Services interoperate with the wider YaCy network over DHT.

This is not a port of the full Java YaCy peer. Scope follows the author's own
needs, not parity with the reference implementation — except where a service
must speak an existing protocol or API, where it stays compliant.

## Examples

Each directory under `examples/` is a runnable compose stack that wires
services into one working deployment.

## Services

Each directory under `services/` is one deployable service. Its
`doc/specification.md` covers its scope, requirements, and known
limitations; its `doc/configuration.md` covers its operator-facing settings.

## Plugins

Each directory under `plugins/` is code loaded into third-party software and
running inside it, not a service of its own. It carries the same two documents.
