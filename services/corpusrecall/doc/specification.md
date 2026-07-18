# corpusrecall — Technical Specification

## Context

`corpusrecall` is a standalone, optional, disposable Go service that yields an operator's own
corpus representation of a requested URL, crawling on demand to fill and refresh it. It
contacts no origin, so a stack whose corpus already holds a page keeps serving it with the
internet down.

The service is protocol-agnostic: it exposes on-demand corpus retrieval behind a narrow
port. An edge adapter translates an external API to that port without changing retrieval.

## Non-Goals

* Fetching, rendering, parsing, or extracting pages — that is `yacycrawler`'s job.
* Holding a corpus of its own or writing to the search index.
* Ranking, reranking, or curating what the index returns.
* Serving the live web.

## Functional Requirements

* The service SHALL accept a request naming one URL and the representations wanted for it.
* The service SHALL ask for a fresh crawl of the URL on every request.
* If the requested URL redirects, the service SHALL return the page for the URL it redirects
  to.
* The service SHALL return each wanted representation the corpus holds by the deadline.
* The service SHALL tell the requester which wanted representations it could not provide.
* The service SHALL let operators configure the search index, the broker, and the retrieval
  deadline.

## Non-Functional Requirements

* The service SHALL persist no state of its own between requests.
* The service SHALL keep memory bounded independently of request volume, with explicit
  limits on in-flight requests and response size.
* The search index and the broker SHALL each sit behind a narrow interface, replaceable with
  no change to retrieval.
* The retrieval port SHALL admit more than one edge adapter over one running core.
* Operational behavior SHALL be observable through machine-readable metrics.

## Known Limitations

* First retrieval of an uncrawled page takes a full crawl-and-index cycle, bounded by the
  deadline.
* A page the crawler declines or cannot reach never enters the corpus, so its retrieval
  fails at the deadline.
