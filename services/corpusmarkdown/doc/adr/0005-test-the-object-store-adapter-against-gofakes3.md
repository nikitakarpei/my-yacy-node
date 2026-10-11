# 5. Test the object store adapter against gofakes3

Status: accepted

## Context

The S3 adapter of `corpusmarkdown` needs an S3 server in its tests. A real store runs in a
container, which `make verify` does not start. The NATS adapters use an in-process server.

## Decision

Tests of the S3 adapter and of the service assembly run `github.com/johannesboyne/gofakes3`
with its in-memory backend, in the test process. The e2e suites run a real Garage node.

## Consequences

- `make verify` tests the adapter through its exported surface without a container.
- The fake can differ from a real store in edge cases; the e2e suites cover the real one.
- The dependency is used only by tests.
