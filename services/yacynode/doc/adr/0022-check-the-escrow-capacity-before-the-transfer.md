# 22. Check the escrow capacity before the transfer

Date: 2026-10-10

## Status

Accepted

Amends ADR 0018.

## Context

The node checked the escrow capacity for each new posting, inside the
transaction that holds the postings of a transfer. That check reads how many
postings the escrow holds. Each transaction that holds, releases or expires a
posting changes that number.

The node will run write transactions at the same time and check, at commit, that
no other transaction changed what one read. With the check inside, two transfers
with unknown URLs always conflict.

## Decision

The node reads the escrow occupancy once, before the transaction of a transfer.
When the escrow is full and the transfer has a posting with an unknown URL, the
node refuses the whole transfer, as ADR 0018 records. Otherwise the escrow holds
all the postings of the transfer, also past the capacity.

## Consequences

The capacity is a soft limit. Transfers that start at the same time below the
capacity can take the escrow past it, by at most their postings.

A transfer no longer reads the escrow occupancy inside its transaction, so
transfers do not conflict on it.
