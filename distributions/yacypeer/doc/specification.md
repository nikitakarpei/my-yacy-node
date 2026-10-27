# yacypeer — Technical Specification

## Context

A YaCy peer of this repository is a set of services that share a NATS server and an egress
proxy. `yacypeer` runs these services, their NATS server, and their egress proxy as one
program, connected and ready to use.

## Non-Goals

* Serving a search page; SearXNG stays a separate program.
* Running a part of the peer alone.

## Functional Requirements

* The program SHALL run `yacynode`, `yacydhtsearch`, `corpustext`, `yacycrawler`,
  `pagescrape`, `crawlconsole`, `crawlscrapebridge`, and `visitcrawl`.
* The program SHALL run a NATS server that only the local host can reach.
* The program SHALL let only the local host reach its endpoints, except the peer endpoint
  of `yacynode`, until the operator sets their addresses.
* The program SHALL NOT send outbound requests to loopback or private addresses.
* The program SHALL send outbound requests through the egress proxy that the operator
  names, also when the operator names a loopback or private IP address for it.
* The program SHALL NOT start when the operator names an egress proxy and does not set
  the public host of the peer.
* The program SHALL keep its local data in one data directory.
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
* The program SHALL name the service in each log line that the service writes.
* The program SHALL stop when one of its services stops.

## Known Limitations

* The peer does not accept postings from other peers.
* Some log lines do not name a service, for example the lines that a library writes.
* The dashboards draw state timeline panels as line charts.
* A crash loses the most recent part of the metric history.
* The program cannot reach an egress proxy that the operator names by a host name with a
  loopback or private address.
* For outbound HTTPS requests, the program follows `NO_PROXY` from its environment. When
  the operator names no egress proxy, it also follows `HTTPS_PROXY`.
