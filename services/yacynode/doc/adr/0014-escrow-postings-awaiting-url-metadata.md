# 14. Escrow inbound postings until their URL metadata arrives

Date: 2026-08-01

## Status

Accepted

## Context

A peer sends postings with `transferRWI` and their URL metadata with a second call, `transferURL`. The
node stored every inbound posting at once. A peer that never made the second call left the node with a
posting for a URL it holds no metadata for.

Such a posting is dead weight. Search ranks it with the rest and then finds no metadata row for it, so it
takes a result slot and returns nothing in it. Eviction cannot remove it either, for the reason ADR 0008
records under known limitations: candidates come from the URL metadata staleness order, and a posting
whose URL has no metadata row is not in that order.

The set only grows. It holds quota that eviction then takes from live postings instead, and on the live
node it also filled the outbound distribution schedule, which offers each posting again every retry
interval and can never settle one whose metadata it cannot send.

## Decision

Hold an inbound posting outside the RWI index until the URL metadata it names arrives.

`rwiescrow` owns two buckets. `rwi_escrow` keys a held posting by URL hash then word hash, so one prefix
scan finds every posting that waits for a URL. `rwi_escrow_expiry` keys it by hold time then URL hash then
word hash, so key order is hold order. The value is the posting form `rwipostings` publishes, prefixed
with the hold time.

Release is per posting. `rwiescrow` observes URL metadata arrivals, so a held posting joins the index
inside the transaction that stores its URL metadata. A background loop drops postings held longer than
five minutes, and drains every expired posting before it waits for its next tick.

The escrow holds a fixed number of postings. Eviction cannot reclaim a held posting, because it selects
from the URL metadata staleness order, so the escrow needs its own bound. A posting that arrives at the
capacity is dropped.

`rwiadmission` owns the routing decision. It asks the URL directory which URLs are unknown, holds the
postings that name them, and admits the rest. The receipt still names the unknown URLs.

Posting intake moves out of `rwipostings`, which becomes a store over the vault alone. This removes the
dependency on `urlmeta` that would otherwise make the construction order cyclic, because `urlmeta` must be
built after the escrow that observes it.

## Consequences

The index holds only postings the node can serve. `rwiescrow_postings_expired_total` measures the peers
that never complete `transferURL`. `rwiescrow_postings_refused_total` rising means inbound orphans arrive
faster than the hold period retires them.

The postings already in the index stay until the vault is replaced: they are not distinguishable from
postings whose peer is merely slow. A posting the escrow drops, at the capacity or after five minutes, is
offered again by its peer on the next cycle.
