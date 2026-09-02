# pagescrape — Technical Specification

## Context

`pagescrape` owns the scrape pipeline for a stack. The pipeline accepts a scrape request,
reads the requested representation through an operator-supplied proxy, offers it to every
corpus, and carries scrape failures and corpus receipts to the outcome feed for the page.
Scrape requests and page offers pass through NATS JetStream.

## Non-Goals

* Deciding which pages to scrape or when to request the first scrape.
* Discovering links, crawling a site, or applying crawl scope rules.
* Deriving, indexing, or storing corpus values from a page.
* Deciding whether a corpus accepts an offered page.
* Bypassing the configured proxy or enforcing the proxy's egress policy.

## Functional Requirements

* For each valid request, the service SHALL start one read of its fetch URL. If the request
  has no fetch URL, the service SHALL read its page URL.
* A page read SHALL follow redirects and route every origin request through the configured
  proxy.
* A successful read SHALL publish one offered page with the page URL, landed URL, content
  type, response body, and any `X-Robots-Tag` values.
* A redirect SHALL NOT change the page URL that identifies the offered page.
* A read that is refused, rejected, unchanged, oversized, invalid after a redirect, or
  otherwise unsuccessful SHALL publish one final failure on the outcome feed for the page.
* A `429` or `503` response SHALL defer the scrape by its `Retry-After` value. An absent or
  invalid value SHALL defer the scrape by one minute.
* A deferred scrape SHALL become due again until it succeeds or the configured deferral
  window passes.
* A request that forbids deferral SHALL fail on its first deferred read.
* A request SHALL remain pending when its offered page or deferral cannot be accepted by the
  broker.
* The service SHALL NOT discard an undecodable request.
* The outcome feed SHALL report which corpus kept or rejected the page.
* At startup, the service SHALL make the scrape request and page offer streams available with
  the configured retention limits.

## Non-Functional Requirements

* The service SHALL bound every origin read by a deadline and a response-size limit.
* The service SHALL bound concurrent intake and unacknowledged scrape requests.
* Pending and deferred requests SHALL survive a service restart within the configured
  retention limits.
* Instances that share a durable consumer name SHALL divide its requests. They SHALL NOT
  each read every request.
* The service SHALL expose scrape request and outcome announcement metrics over HTTP.

## Known Limitations

* The outcome feed is not durable. A listener that is away when an outcome is sent does not
  receive it later.
* Outcomes are keyed by page URL, not by request. Concurrent requests for the same page URL
  cannot identify which request produced an outcome.
* The service does not publish one outcome that says every corpus has finished.
* The broker does not retain a page offer when no corpus consumer has interest in it.
* One undecodable request stops intake until an operator resolves it.
* An outcome announcement can be lost if publication fails after the request is complete.
* A transient origin failure has the reason `no-reason-given` and carries no detailed cause.
