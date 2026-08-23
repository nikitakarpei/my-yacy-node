# 14. Publish one message per reached page

Date: 2026-08-22

## Status

Accepted

## Context

The crawler publishes each page it reached to `CRAWL_REACHED_PAGES`. A message carries an
order identifier and a canonical URL, about 110 to 200 bytes. The publish frame adds about
40 bytes and the JetStream acknowledgment subject about 100 more, so the overhead is 50 to
100 percent of the payload. That ratio invites batching.

## Decision

The crawler publishes one message for each reached page.

Each message causes up to three fetches of up to `YACYCRAWLER_MAX_BODY_BYTES` in the
consumers. Broker traffic is near 0.005 percent of the traffic it starts, so batching makes
the small term smaller.

One message for each finished order needs a buffer that grows with the size of the run,
which the crawler must not do, and passes the default 1 MiB payload limit near 13000 URLs.
It would need chunking again, and a consumer would see nothing until the order ends.

Batches of a fixed size remove most of the overhead but add a flush timer, a batch limit,
and a rule for what a consumer acknowledges when one URL in the batch fails. One message for
each URL gives exact retry granularity for no extra cost.

## Consequences

A consumer acknowledges each URL on its own and retries only that URL. Batching stays
available later as a change to the contract and to each consumer's intake loop. Examine it
again if broker egress becomes visible in a metric, or if the broker becomes remote or
metered.
