# 4. Keep answering searches while the page offers are unavailable

Status: accepted

## Context

A service fails fast when it cannot open a stream it does not own: it does not
start. `corpustext` reads the page offers stream of `pagescrape`. Since ADR 3,
`corpustext` also answers searches. Failing fast on the page offers would stop
local search whenever the broker of the crawler is down, although the search
index is available.

## Decision

`corpustext` starts to open the page offers in the background before it opens
its search index, so a slow search index does not delay its consumer. It
serves searches without waiting for the page offers. While
the broker or the stream is unavailable, it logs an error and tries again after
a pause that doubles up to 30 seconds. It does not stop.

A search index that cannot be opened still stops the service at start.

## Consequences

- Local search works while the broker is down.
- A wrong broker address no longer stops the service. The operator sees it in
  the error log, not in the exit status.
- Once open, a lost broker connection follows the reconnect rules of the NATS
  client; when the client gives up, intake ends and the service stops.
