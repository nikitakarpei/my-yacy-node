# 2. Address the crawl and the page markdown store separately

Status: accepted

## Context

`corpusmarkdown` speaks to JetStream for two unrelated reasons. It consumes the
crawled page markdown stream, which the crawl fleet owns and which holds pages only
until every consumer acknowledges them. It also writes the page markdown bucket,
which `corpusmarkdown` owns and which holds the corpus for as long as an operator
keeps it. `corpusrecall` reads both the same way.

A single `NATS_URL` bound the two to one server. An operator who wanted to size,
replicate, or back up the corpus differently from the crawl had no way to do it, and
an operator who lost the crawl server lost the corpus with it.

## Decision

Each service that speaks to both takes one endpoint per concern: `CRAWL_NATS_URL` for
the crawl, and `PAGE_MARKDOWN_NATS_URL` for the page markdown bucket. Both are
required, and the service opens one connection to each.

The same URL in both variables puts the two concerns on one server, which stays the
simple deployment. No default couples them, so the reuse is a choice an operator
makes and reads back from the environment.

## Consequences

- An operator sizes and replicates the corpus independently of the crawl.
- The corpus survives a rebuild of the crawl server, and the reverse.
- Every deployment names both endpoints, including the single-server one.
- A service that speaks to both holds two connections instead of one.
