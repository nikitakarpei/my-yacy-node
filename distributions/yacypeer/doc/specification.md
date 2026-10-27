# yacypeer — Technical Specification

## Context

`yacypeer` runs the services of a YaCy peer, their NATS server and their egress proxy as one
program.

## Non-Goals

* Running a part of the peer alone.

## Functional Requirements

* The program SHALL run `yacynode`, `yacydhtsearch`, `corpustext`, `yacycrawler`,
  `pagescrape`, `crawlconsole`, `crawlscrapebridge`, `visitcrawl`, and `searchpage`.
* The program SHALL run a NATS server that only the local host can reach.
* The program SHALL let only the local host reach its endpoints, except the peer endpoint
  of `yacynode`, until the operator sets their addresses.
* The program SHALL NOT send outbound requests to loopback or private addresses, except page
  fetches through a page fetch proxy.
* The program SHALL send requests to peers and seedlists through the peer proxy that the
  operator names, also when the operator names a loopback or private IP address for it.
* The program SHALL send page fetches through the page fetch proxy that the operator names,
  else as it sends requests to peers.
* The program SHALL answer a repeated page fetch from its cache, per page and user agent,
  within the cache retention.
* The cache SHALL hold only final answers to unconditional fetches, within the size limit.
* The services SHALL fetch pages with one user agent.
* The program SHALL NOT start when the operator names a peer proxy and does not set the
  public host of the peer.
* The program SHALL NOT start when `HTTP_PROXY`, `HTTPS_PROXY`, or `NO_PROXY` is set.
* The program SHALL keep its local data in one data directory.
* The search page SHALL search the YaCy network and the crawled pages, and SHALL send a
  person who opens a result through `visitcrawl`.
* The program SHALL make the secret that signs visit links on its first start, and SHALL
  keep it in the data directory.
* The program SHALL wire the settings that connect the services, and SHALL NOT start when
  the operator sets one of them or an ops setting of a service.
* The program SHALL give a setting with a service prefix priority for that service.
* The program SHALL serve the metrics of the process and of each service on the ops address.
* The program SHALL keep a metric history of the process and of each service in the data
  directory for the retention period that the operator sets.
* The program SHALL show the dashboard of each service that has one on the ops address, and
  SHALL draw it from the metric history.
* The search page, the crawl console and the dashboards SHALL link each other in a tab bar.
* The program SHALL name the service in each log line that the service writes.
* The program SHALL stop when one of its services stops.

## Known Limitations

* The peer does not accept postings from other peers.
* Some log lines do not name a service.
* The dashboards draw state timeline panels as line charts.
* A tab links a page that listens on every address through the local host. Other hosts
  cannot follow that tab.
* A crash loses the most recent part of the metric history.
* The program cannot reach a peer proxy that the operator names by a host name with a
  loopback or private address.
* Within the cache retention, a page fetch gets the cached page, also when the page changed.
* A proxy that applies rules to each request does not see the page fetches that the cache answers.
