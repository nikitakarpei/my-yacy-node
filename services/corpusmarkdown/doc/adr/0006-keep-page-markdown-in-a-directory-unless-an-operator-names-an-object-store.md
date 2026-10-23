# 6. Keep page markdown in a directory unless an operator names an object store

Status: accepted

## Context

[ADR 4](0004-store-page-markdown-in-an-s3-compatible-object-store.md) made an
S3-compatible object store necessary to start the service. An operator with a small
corpus had to run, secure, and back up a second server for it.

## Decision

Without an object store endpoint, the service keeps the markdown of each page in a file
in a directory of the local filesystem. It uses only the Go standard library.

The file holds only the markdown. The modification time of the file is the time the
service stored it. The service calculates the version from the markdown when it reads
the file. A subdirectory, named by the first two characters of the object name, holds
each file.

The service writes the markdown to a pending file, and then renames the pending file to
the file name. With an endpoint, the service uses the object store of ADR 4. An endpoint
and a directory together are a configuration error.

## Consequences

- The service starts with no storage server.
- The service module gets no new dependency.
- Instances on one host can share a directory. Instances on different hosts need an
  object store.
- The service does not move markdown from a directory to an object store. A new crawl
  fills the new store.
- The stack chapters keep the markdown in a volume. A separate chapter runs Garage.
