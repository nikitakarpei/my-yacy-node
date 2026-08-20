# 1. Store page markdown in the NATS JetStream Object Store

Status: accepted

## Context

`corpusmarkdown` keeps the latest markdown representation of every crawled page,
addressed by the page's canonical URL, so an exact-URL reader can recall a page
verbatim. A page's markdown ranges from a few kilobytes to a few megabytes, and a
re-crawl must overwrite the prior copy rather than accumulate versions.

The node already runs NATS JetStream as its only broker and durable-storage
dependency; the crawled-page representations this service consumes arrive over it.
JetStream ships an Object Store keyed by name with overwrite-by-name semantics and
no per-value size ceiling, which is the exact shape this store needs.

An external object store such as S3 or MinIO offers the same key-addressed,
overwrite-by-name model and scales further, at the cost of a second piece of
infrastructure for operators to run, secure, and back up.

## Decision

Store each page's markdown in a JetStream Object Store bucket, under an object
name derived from the canonical URL. Re-crawling a URL puts to the same name and
replaces the prior markdown, so the store holds one current copy per URL.

The URL-to-object binding lives in one place, `pagemarkdownstore`, imported by the
writer and any future reader so both address a page identically.

## Consequences

- The service needs no storage technology beyond the JetStream the node already runs.
- Recall is a single get by canonical URL; no query engine or index is involved.
- Object Store durability is bounded by the JetStream deployment's storage and
  replication; operators size and replicate JetStream for the corpus they keep.
- Moving to S3 or MinIO later is a new adapter behind the `pagemarkdownstore` seam,
  not a change to the writer or reader.
