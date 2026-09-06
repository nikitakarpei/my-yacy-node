# pagescrape — Technical Specification

## Context

`pagescrape` owns the scrape pipeline for a stack. The pipeline accepts a scrape request,
fetches the page content through an operator-supplied proxy, offers it to every corpus, and
carries scrape failures and corpus receipts to the outcome feed for the page.
Scrape requests and page offers pass through NATS JetStream.

## Glossary

* **Page URL** — the URL that identifies the page.
* **Fetch URL** — the URL the service requests the content from. The page URL when the
  request names none.
* **Landed URL** — the URL of the last response of the fetch after all redirects.

## Non-Goals

* Deciding which pages to scrape or when to request the first scrape.
* Discovering links, crawling a site, or applying crawl scope rules.
* Deriving, indexing, or storing corpus values from a page.
* Deciding whether a corpus accepts an offered page.
* Bypassing the configured proxy or enforcing the proxy's egress policy.

## Functional Requirements

* For each valid request, the service SHALL make one fetch, at the fetch URL of the request.
* The service SHALL send every origin request through the configured proxy.
* A fetch SHALL follow redirects, but no more hops than the configuration permits.
* A redirect SHALL NOT change the page URL that identifies the offered page.
* A fetch that succeeds SHALL offer the page one time. The offer carries the page URL, the
  landed URL, the content type, the content, and all `X-Robots-Tag` values.
* A fetch that does not succeed SHALL put one last failure on the outcome feed for the page.
* The service SHALL defer the scrape when the origin answers `429` or `503`. It SHALL wait
  for the time in `Retry-After`, or one minute when that time is absent or unreadable.
* The service SHALL scrape a deferred request again, until it succeeds or until the
  configured deferral window ends.
* A request that forbids deferral SHALL fail as soon as the origin defers it.
* A request SHALL stay pending when the broker does not accept its offered page or its
  deferral.
* The service SHALL NOT discard a request it cannot decode.
* The outcome feed SHALL tell which corpus kept the page and which corpus rejected it.
* At startup, the service SHALL create the scrape request stream and the page offer stream
  with the configured retention limits.

## Non-Functional Requirements

* The service SHALL stop an origin fetch that goes past its deadline or its size limit.
* The service SHALL limit how many requests it takes at the same time, and how many stay
  unacknowledged.
* Pending and deferred requests SHALL stay after a service restart, within the configured
  retention limits.
* Instances that share a durable consumer name SHALL divide the requests. Each request SHALL
  go to one instance only.
* The service SHALL supply scrape request and outcome announcement metrics over HTTP.

## Known Limitations

* The outcome feed is not durable. A listener that is away when an outcome is sent does not
  receive it later.
* Outcomes are keyed by page URL, not by request. When two requests for the same page URL
  run together, a listener cannot tell which request made an outcome.
* The service does not publish one outcome that says every corpus has finished.
* The broker does not retain a page offer when no corpus consumer has interest in it.
* A request the service cannot decode stops intake until an operator resolves it.
* An outcome announcement can be lost if publication fails after the request is complete.
* A transient origin failure has the reason `no-reason-given`. It carries no cause.
