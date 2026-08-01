# 13. Batch URL metadata delivery to peers

Date: 2026-08-01

## Status

Accepted

## Context

A peer's DHT posting offer is followed by a URL metadata delivery step: for every offered posting whose
URL the peer does not yet recognize, this node sends full URL metadata (title, snippet, tags, and related
fields) so the peer can serve that posting in search results. That step, `transferURL.html` upstream,
sends every unknown URL as one HTTP POST with one form field per URL row, alongside four fixed fields.

That single request can carry as many rows as the RWI offer step accepted in one cycle, bounded
incidentally by `postingsPerPeerCap` (1000, matching upstream's own `Transmission.maxRWIsCount`). Upstream
YaCy's own HTTP server (`YaCyHttpServer.java`) serves that endpoint under Jetty's default form limits:
200,000 bytes per request body and 1,000 form keys. A worst-case row — long title, tag list, and
snippet, all base64-encoded — can push a full 1000-row request well past the byte limit long before it
reaches the key-count limit, causing Jetty to reject the whole request and the peer to receive none of the
offered metadata for that cycle. This is a latent gap in the reference YaCy implementation itself: its
`transferRWI` step has both a client-side cap (`Transmission.maxRWIsCount`) and a server-side backstop
(`transferRWI.java`, `count > 1000 break`), but `transferURL` has neither.

`transferURL`'s server-side handler is stateless and idempotent per request: re-sending a URL that a peer
already has just increments a "double" counter, with no error. It has no server-side cap on row count.
This makes it protocol-legal, and safe, to split one logical delivery across several POSTs to the same
peer.

## Decision

Split URL metadata delivery into fixed-size batches at the transport seam. A new
`boundedURLMetadataCourier` decorator wraps the existing `httpURLMetadataCourier`, both implementing the
same `urlMetadataCourier` interface, so nothing above that seam changes: `postingOfferDelivery` still
makes one logical `Deliver` call per peer per cycle.

The decorator sends batches of `URLMetadataBatchSize` rows in order. If a batch is accepted, its per-URL
rejections are accumulated and the next batch is sent. If a batch comes back deferred, refused, or
unreachable, remaining batches for that peer this cycle are not sent, and the whole delivery is reported
as failed, even for URLs whose batches already succeeded. Those URLs' postings are excluded from this
cycle's accepted set and retried at the plain retry interval, same as today; because `transferURL` is
idempotent, a retry only re-sends metadata the peer may already hold, at the cost of one redundant POST,
not a correctness risk.

`URLMetadataBatchSize` is a new `rwidistribution.Config` field, defaulting to 50, derived from Jetty's
200,000-byte request limit and a rough worst-case per-row size estimate (~2,000-2,500 bytes, dominated by
base64-encoded title/snippet/tags/publisher/favicon fields) with roughly 1.6x margin against that
estimate, since no measured row-size sample exists in this codebase.

## Considered alternatives

Shrinking `postingsPerPeerCap` (the RWI offer cap) instead of batching URL metadata was rejected: that
constant is specific to the RWI offer step, already matches upstream's own cap exactly, and is transmitted
as one opaque blob field immune to the per-key and byte limits that only affect the separate, per-row-keyed
URL metadata step. Coupling the two would tie an unrelated posting-offer throughput concern to a
URL-metadata wire-size concern.

Sending all rows in one request regardless of size, and letting Jetty reject oversized requests, was
rejected: a rejection discards the entire batch's metadata, including rows that would have fit comfortably,
for no benefit over sending several smaller requests that each fit.

Splitting inside `postingOfferDelivery.deliverURLMetadata` instead of a courier decorator was rejected:
transport sizing is not a policy concern, and a decorator on the existing `urlMetadataCourier` seam keeps
the retry and accept-filtering logic in `postingOfferDelivery` completely unchanged and independently
testable from the batching logic.

Continuing remaining batches after a failed batch was rejected: a deferred, refused, or unreachable
outcome is a peer- or transport-level signal that does not change batch to batch within one cycle, so
retrying further batches immediately would spend extra round trips to re-learn the same fact, instead of
simply waiting for the next cycle.

## Known limitations

The 50-row default is derived from an estimated, not measured, worst-case row size; a future peer
population with unusually large metadata fields could still exceed 200,000 bytes at that batch size.
There is no runtime measurement of actual encoded batch size as a cross-check, and no automatic shrinking
of the batch size in response to a peer's byte-limit rejection this cycle, only the next cycle's retry.

## Consequences

Peers with many unknown URLs in one cycle now reliably receive all of their metadata across several
smaller requests, instead of one oversized request that a strict Jetty configuration could reject
outright. Operators can tune `URLMetadataBatchSize` via `YACY_DISTRIBUTION_URL_METADATA_BATCH_SIZE` if a
peer population needs a smaller or larger value. The RWI offer step and its cap are untouched.
