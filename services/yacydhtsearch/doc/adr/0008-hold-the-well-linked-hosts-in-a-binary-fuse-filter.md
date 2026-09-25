# 8. Hold the well-linked hosts in a binary fuse filter

Date: 2026-09-25

## Status

Accepted

## Context

Spam in the results comes mostly from hosts that few other hosts link to. On the judged
queries, results of hosts outside the ten million best-linked hosts of a Common Crawl host
graph ranked lower cut the spam among the first ten results and kept the gain. A list of ten
million hosts takes about one gigabyte as a Go map.

## Decision

When an operator names a list of well-linked hosts, the service reads it at start and holds
it in a binary fuse filter from `github.com/FastFilter/xorfilter`, under the Apache 2.0
license, which carries no dependencies of its own. The service builds the filter itself, so
the file stays a plain list that any source can give.

## Consequences

Ten million hosts take about 11 megabytes and add about one second to the start. Fewer than
one in two hundred hosts outside the list are held by mistake; their results keep the rank
they have without the list. Another source of hosts gives another rule, which the judged
queries must accept before an operator uses it.
