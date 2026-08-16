# 18. Configure the escrow capacity and refuse when it is full

Date: 2026-08-16

## Status

Accepted

Amends ADR 0014.

## Context

Eviction cannot reclaim a held posting, for the reason ADR 0008 records. The
escrow therefore needs its own bound.

ADR 0014 derived that bound from the storage quota, to stop a peer from filling
the disk. The escrow holds only a posting whose URL is unknown, so a peer that
sends the URL metadata with its postings bypasses the escrow. The bound also
guards approximately 54 megabytes on a node with a one gigabyte quota, and
production metrics show the escrow holds one percent of the bound or less.

At the bound, the escrow dropped the new posting but the node answered that the
transfer was satisfactory. A sender removes a posting from its own index when
the transfer succeeds. The node thus destroyed the last copy, and the sender did
not know.

## Decision

The operator configures the capacity of the escrow, as a number of postings. The
default is 8192 postings. The escrow no longer reads the storage quota.

When the escrow is full, the node refuses the transfer and reports that it is
busy. The sender keeps its postings and offers them again later. Only the end of
the hold period frees space.

## Consequences

The node loses no posting at the limit. It refuses a whole request and keeps no
posting of that request.

A YaCy sender marks a node that reports busy as a node that does not accept
index transfers, until it reads the seed of that node again.

The occupancy panel now measures the held postings against a limit the operator
sets. An operator who sees refusals raises the limit.
