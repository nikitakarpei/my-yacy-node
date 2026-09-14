# 7. Discount the relevance of a stale document

Date: 2026-09-14

## Status

Accepted

## Context

The ordering of the service answered a query from the text of a document only. A page that
answered the query in 2011 held the same place as one that answers it today. Peers send the
day a document was modified on both paths that give the service its answers.

## Decision

The relevance of each answered document is multiplied by the share it keeps at its age. Half
of the relevance is at stake to the age, and the document keeps half of that share for each
year since the day the peers say it was modified. The share therefore stays between half and
one.

The discount reads the modified day only. A document the peers do not date, or date on the day
of the search or after it, keeps its whole relevance. The day of the search is read once per
query.

## Consequences

A fresher document comes before an older document of the same text. A document can lose at
most half of its relevance to age, so a strong old document still comes before a weak fresh
one, and a peer that sends a wrong day moves a document by a bounded amount.

A document the peers do not date counts as fresh. A peer that dates a document by the day it
was crawled therefore makes that document look as fresh as the crawl. This favours a recently
crawled document over an honestly dated old one.

The discount runs before the host discount, and before the service chooses the pages it reads,
so it changes which pages one query reads.
