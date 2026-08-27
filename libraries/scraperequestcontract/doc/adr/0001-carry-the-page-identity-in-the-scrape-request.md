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

A search result must name the page as the world knows it, not the copy one
deployment holds.

The producer knows both facts. An archive index gives the capture address and
the page URL in one record. A request with one URL discards the other.

## Decision

The scrape request names two facts:

* the URL that identifies the page, under which the corpus indexes it
* the URL from which the corpus reads the bytes

The second fact is optional. When the request does not give it, the corpus reads
the page from its own URL. A live-web request stays as it is.

The producer states both facts. The consumer reads from one address and indexes
under the other. It takes the identity from the response only when the two
addresses are the same.

The corpus follows a redirect at the read address.

## Considered alternatives

To let the consumer ask the copy which page it holds is rejected. The answer can
be absent or wrong, and the page then enters the index under an address that
identifies nothing.

To translate the identity into a read address inside the read path is rejected.
It commits a whole deployment to one source. A node cannot then index live pages
and archived pages together.

## Consequences

A corpus indexes an archived page under the URL a person knows, and the search
result leads to that page.

Any source that serves a page from an address of its own can now feed the index.
A mirror and a cached copy need no new vocabulary.

The producer resolves a redirect that an archive captured, and names the page it
found.

Nothing in the read path knows what an archive is.

Nothing verifies that the bytes at the read address are the named page. The
producer is the only guard of that promise.

A redirect on the live web can put a page in the index that no producer named.
The read lands at another address, and that address becomes the identity.

An older producer stays correct, because a request without the second fact keeps
its meaning.
