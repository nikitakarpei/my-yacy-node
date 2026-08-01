A lightweight, composable reimplementation of a [YaCy](https://yacy.net) peer
as small Go services. Each service runs and scales on its own, and a
deployment wires together only the services it needs. Services interoperate
with the wider YaCy network over DHT.

## Motivation

A stock YaCy node exceeds the memory of small hardware such as a Raspberry
Pi, and the excess grows with the corpus. That constraint started this
project, which has grown into an experiment in an alternative design for
YaCy. It runs in the author's homelab and serves that use first, with no
goal of full compatibility with the original YaCy.

## Examples

Each directory under `examples/` is a runnable compose stack that wires
services into one working deployment.

## Documentation

Each service under `services/` and each plugin under `plugins/` carries its own
`doc/`: `specification.md` for scope, requirements, and known limitations;
`configuration.md` for operator-facing settings.
