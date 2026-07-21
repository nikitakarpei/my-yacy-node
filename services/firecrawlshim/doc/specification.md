# firecrawlshim — Technical Specification

## Context

`firecrawlshim` is an edge adapter that exposes the Firecrawl v1 scrape API over HTTP and
translates each call into a corpusrecall recall. It lets a client written against Firecrawl
retrieve an operator's own corpus representation of a URL without changing that client.

## Non-Goals

* Fetching, rendering, parsing, or extracting pages — corpusrecall and yacycrawler do that.
* Holding a corpus or any state of its own.
* Reproducing Firecrawl features beyond single-URL scrape.

## Functional Requirements

* The service SHALL accept `POST /v1/scrape` naming one URL and the formats wanted for it.
* The service SHALL map each supported format to a corpusrecall representation kind and
  default to markdown when none is named.
* The service SHALL return the recalled representation in the Firecrawl scrape response
  shape, carrying the page's canonical URL as its source URL.
* The service SHALL reject a request that names no URL or that it cannot decode.
* The service SHALL report a failed recall to the caller as a gateway error.

## Non-Functional Requirements

* The service SHALL persist no state of its own between requests.
* The corpusrecall target SHALL be configurable, and the recall wait SHALL be bounded.

## Known Limitations

* Only the markdown and text formats map to a representation; other Firecrawl formats yield
  no content.
* A representation the corpus does not hold by the recall timeout is absent from the
  response.
