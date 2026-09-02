# 2. Keep the page identity the request named

Date: 2026-09-02

## Status

Accepted

## Context

ADR 1 gives a scrape request two URLs: the URL that identifies the page, and the
URL the service reads the bytes from. It also gives the service a rule for the
usual request, where the two URLs are the same: if the read follows a redirect,
the landing becomes the identity of the page.

That rule moves the identity of a page after a caller asked for it. The service
offers the page under the landing. Each corpus stores it under the landing and
sends its receipt on the feed of the landing. A caller that waits on the URL it
asked for hears nothing, and its recall of that URL finds nothing.

The service kept a bucket of redirections so that a caller can find the landing.
The bucket does not repair the wait. It is empty before the scrape, and the wait
is already over after it.

## Decision

The page keeps the identity the request named. A redirect does not change it.

An offered page holds two URLs:

* the URL that identifies the page, which each corpus stores it under and
  answers on
* the URL the read landed on, which is the base address for the links in the
  document

The service records no redirection, and the bucket of redirections is dropped.

## Considered alternatives

To carry the URL that was asked for beside the identity is rejected. It adds a
third URL to say what the first one already says.

To let the outcome feed translate a landing back to the URL that was asked for
is rejected. It needs an index from the landing to every URL that reaches it,
and a read of that index on every receipt.

## Consequences

A caller asks for a URL, waits on that URL, and recalls that URL. Nothing
between the caller and the corpus resolves a redirection.

Two URLs that redirect to one page are stored twice, under both identities. The
bytes are the same, and the store holds one copy per identity.

A recall of the landing finds nothing until somebody asks for the landing. The
crawler is not affected: it follows the redirect during the visit and asks to
scrape the URL it settled on.

A page that redirects to a page outside the scope of the operator enters the
corpus under the URL that was asked for, so no landing enters the corpus that
no caller named.
