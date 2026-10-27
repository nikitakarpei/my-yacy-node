# 7. Answer repeated page fetches from page copies

Status: accepted

## Context

A search reads a page, a click on the result crawls it, and the scrape fetches it again. In
`yacypeer` the origin can serve one page three times in a few minutes. Most pages forbid
caching in their headers, and an `https` fetch through a tunnel hides the page from a proxy.

## Decision

Run a forward proxy inside `yacypeer` that the page fetchers reach with absolute URLs. It keeps
each answer as one file per user agent and URL for a fixed retention, whatever the cache
headers of the page say, and a sweep removes old files within a byte maximum.

## Consequences

* One origin fetch serves the page read, the crawl and the scrape of a page.
* A changed page is served from its copy until the retention ends.
* The proxy needs no database, and its memory holds only the answers in transit.
* An operator can send page fetches through a proxy of their own, apart from peer requests.
