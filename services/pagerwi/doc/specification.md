# pagerwi — Technical Specification

## Context

`pagerwi` builds the reverse word index (RWI) of each page that `pagescrape` offers. The RWI
of a page holds one posting for each word of the page and one URL metadata row with the
postings root. The service publishes the RWI of each page to the nodes. Page offers and RWIs
pass through NATS JetStream.

## Glossary

* **Partition exponent** — the DHT ring partition exponent of the YaCy network. The postings
  root changes with it.

## Non-Goals

* Fetching pages or deciding which pages to scrape.
* Storing an RWI after a node took it.
* Sending postings to YaCy peers.

## Functional Requirements

* The service SHALL build one RWI for each offered page that holds a document it reads.
* The RWI SHALL identify the page by the page URL of the offer, not by the landed URL.
* The service SHALL NOT make a posting for a stopword of the page language.
* The service SHALL build each RWI for the configured partition exponent.
* The service SHALL publish the RWI of each page one time, on the RWI stream.
* The service SHALL report a page as kept when the stream takes its RWI.
* The service SHALL report a page as rejected when it holds no document, has a media type
  the service does not read, or gives no text.
* An offered page SHALL stay pending when the stream does not take its RWI.
* The service SHALL NOT discard an offered page it cannot decode.
* At startup, the service SHALL create the RWI stream with the configured size limit.
* The RWI stream SHALL keep an RWI until every node consumer took it, and SHALL refuse a new
  RWI while it is full.
* The service SHALL take no new offered page while a durable consumer of the RWI stream has
  more unfinished RWIs than the backlog limit.

## Non-Functional Requirements

* The service SHALL hold no state of its own.
* The service SHALL limit how many offered pages it works on at the same time.
* Instances that share a durable consumer name SHALL divide the offered pages.
* While the page offers are unavailable, the service SHALL try again after a pause that grows
  with each try.
* The service SHALL supply page intake and intake receipt metrics over HTTP.

## Known Limitations

* An RWI that no node consumer has interest in is lost, but the page is reported as kept.
* A node stops its intake when its partition exponent is not the one of the service.
* An offered page the service cannot decode stops intake until an operator resolves it.
* A durable RWI consumer that no node reads stops intake until an operator removes it.
