# 1. Carry the page identity in the scrape request

Date: 2026-08-27

## Status

Accepted

## Context

A scrape request asks a corpus to read a page and to put it in the index. The
request holds one URL. That URL is the address the corpus reads, and it is also
the identity the corpus indexes.

The two are one fact for a page on the live web. They are two facts for other
sources. A web archive serves a capture of a page at an address of its own. The
bytes belong to the page. The address belongs to the archive.

Identity is what the index promises. A search result must name the page as the
world knows it, not as one deployment holds a copy. An address that only one
archive resolves has no value to the person who reads the result.

The producer of the request knows both facts together. An archive index gives
the address of a capture and the URL of the page in one record. A request that
carries one of them discards the other.

A consumer that must find the identity again can only ask the copy what page it
is. This makes the index trust an answer that no party authenticates, and the
answer can be absent.

## Decision

The scrape request names two facts:

* the URL that identifies the page, under which the corpus indexes it
* the URL from which the corpus reads the bytes

The second fact is optional. When the request does not give it, the corpus reads
the page from its own URL. A live-web request stays as it is.

The producer states both facts. The consumer reads from one address and indexes
under the other. No consumer takes the identity from the response.

The corpus reads the bytes in one step. It does not follow a redirect to a
different page. The producer names each page of a run, so the run does not
discover pages while it reads them.

## Considered alternatives

To let the consumer recover the identity from the response is rejected. The
producer holds the fact and gives it away, and only the source can give it back.
Recovery can fail, and the page then enters the index under an address that
identifies nothing.

To translate the identity into a read address inside the read path is rejected.
It commits a whole deployment to one source. A node cannot then index live pages
and archived pages together.

## Consequences

A corpus indexes an archived page under the URL a person knows, and the search
result leads to that page.

Any source that serves a page from an address of its own can now feed the index.
A mirror and a cached copy need no new vocabulary.

The producer owns redirect resolution. A producer that wants the target of a
capture-time redirect resolves it in its own index and names the page it found.

Nothing in the read path knows what an archive is.

Nothing verifies that the bytes at the read address are the named page. The
producer is the only guard of that promise.

An older producer stays correct, because a request without the second fact keeps
its meaning.
