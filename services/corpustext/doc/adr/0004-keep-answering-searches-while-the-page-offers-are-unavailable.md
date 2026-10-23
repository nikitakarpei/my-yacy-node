# 4. Keep answering searches while the page offers are unavailable

Status: accepted

## Context

A service that cannot open a stream it does not own fails fast. Since ADR 3,
`corpustext` also answers searches, so failing fast on the page offers of
`pagescrape` would stop local search whenever the broker is down.

## Decision

`corpustext` serves searches without waiting for the page offers. While the broker or
the stream is unavailable, it logs an error and tries again, with a pause that doubles
up to 30 seconds. A search index that cannot be opened still stops the service at start.

## Consequences

- Local search works while the broker is down.
- A wrong broker address shows in the error log, not in the exit status.
- When the NATS client gives up reconnecting to a broker it had reached, the service
  stops.
