# 13. Batch URL metadata delivery to peers

Date: 2026-08-01

## Status

Accepted

## Context

After a peer accepts a posting offer, the node sends full metadata for every URL the peer reports unknown.
That step, `transferURL.html`, sent all of them as one HTTP POST with one form field per URL row.

Upstream YaCy serves that endpoint under Jetty's default form limits: 200,000 bytes per body and 1,000
form keys. One cycle can offer up to 1,000 postings, and a worst-case row passes the byte limit long
before the key limit, so Jetty rejects the whole request and the peer receives none of the metadata.

The handler is stateless and idempotent per request: a URL the peer already holds only increments a
counter. Splitting one logical delivery across several POSTs is therefore protocol-legal.

## Decision

Split delivery into fixed-size batches at the transport seam. `boundedURLMetadataCourier` decorates
`httpURLMetadataCourier`, both implementing `urlMetadataCourier`, so `postingOfferDelivery` still makes one
`Deliver` call per peer per cycle.

The decorator sends batches in order and accumulates their per-URL rejections. If a batch is deferred,
refused, or unreachable, the remaining batches are not sent and the whole delivery is reported as failed,
including URLs whose batches already succeeded. Those postings are retried at the plain retry interval.
Such an outcome is a peer- or transport-level signal that does not change between batches within one cycle.

`URLMetadataBatchSize` defaults to 50, derived from the 200,000-byte limit and an estimated worst-case row
of 2,000 to 2,500 bytes with roughly 1.6x margin.

## Considered alternatives

Shrinking `postingsPerPeerCap` was rejected: it is specific to the RWI offer step, already matches
upstream's cap, and that step sends one opaque blob field immune to the per-key and byte limits.

Splitting inside `postingOfferDelivery` was rejected: transport sizing is not a policy concern, and the
decorator keeps retry and accept-filtering independently testable.

## Consequences

Peers with many unknown URLs receive all of their metadata across several smaller requests. Operators tune
`URLMetadataBatchSize` through `YACY_DISTRIBUTION_URL_METADATA_BATCH_SIZE`.

The default rests on an estimated worst-case row size. Nothing measures the encoded batch size at runtime.
Reporting already-delivered URLs as failed costs those postings a full retry interval whenever a later
batch fails.
