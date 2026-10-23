# 4. Store page markdown in an S3-compatible object store

Status: accepted, amended by [ADR 6](0006-keep-page-markdown-in-a-directory-unless-an-operator-names-an-object-store.md)

## Context

`corpusmarkdown` kept the markdown of each crawled page in a NATS JetStream object store.
Each object takes its own subjects, which the broker holds in memory, so broker memory
grows with the corpus. This makes the stack harder to run on hardware with little memory.

## Decision

The service keeps the markdown in a bucket of an S3-compatible object store and creates
that bucket at start. It talks to the store with `github.com/aws/aws-sdk-go-v2/service/s3`,
which works with AWS S3 and with self-hosted stores such as Garage.

## Consequences

- Broker memory no longer grows with the corpus. NATS only carries messages.
- Operators run an S3-compatible store; the stack chapters run a single Garage node.
- Compression on disk is the duty of the store, not of the service.
- The SDK adds a large dependency tree to the service module.
