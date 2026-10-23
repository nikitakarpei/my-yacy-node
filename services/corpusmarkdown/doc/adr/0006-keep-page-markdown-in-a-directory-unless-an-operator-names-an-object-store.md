# 6. Keep page markdown in a directory unless an operator names an object store

Status: accepted

## Context

[ADR 4](0004-store-page-markdown-in-an-s3-compatible-object-store.md) made an
S3-compatible object store necessary to start the service. An operator with a small
corpus had to run and back up a second server for it.

## Decision

Without an object store endpoint, the service keeps the markdown of each page in a file
in a local directory. The modification time of the file is the time the service stored
it. With an endpoint, the service uses the object store of ADR 4. An endpoint and a
directory together are a configuration error.

## Consequences

- The service starts with no storage server and no new dependency.
- Instances on different hosts need an object store; a directory serves one host.
- Markdown does not move from a directory to an object store. A new crawl fills it.
