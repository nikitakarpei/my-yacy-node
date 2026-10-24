# yacypeer — Technical Specification

## Context

A YaCy peer of this repository is a set of services that share a NATS server and an egress
proxy. `yacypeer` runs these services, their NATS server, and their egress proxy as one
program, connected and ready to use.

## Non-Goals

* Merging or ranking the results of different services; SearXNG stays a separate program.
* Showing metrics; Prometheus and Grafana stay separate programs.
* Running a part of the peer alone.

## Functional Requirements

* The program SHALL run `yacynode`, `yacydhtsearch`, `yacydhtsearchpage`, `corpustext`,
  `yacycrawler`, `pagescrape`, `crawlconsole`, `crawlscrapebridge`, and `visitcrawl`.
* The program SHALL run a NATS server that only the local host can reach.
* The program SHALL let only the local host reach its endpoints, except the peer endpoint
  of `yacynode`, until the operator sets their addresses.
* The program SHALL send outbound requests through the egress proxy that the operator
  names, else through a built-in egress proxy that only the local host can reach.
* The program SHALL NOT start when the operator names an egress proxy and does not set
  the public host of the peer.
* The program SHALL keep its local data in one data directory.
* The program SHALL make the secret that signs visit links on its first start, and SHALL
  keep it in the data directory.
* The program SHALL wire the settings that connect the services, and SHALL NOT start when
  the operator sets one of them or an ops setting of a service.
* The program SHALL give a setting with a service prefix priority for that service.
* The program SHALL serve the metrics of the process and of each service on one address.
* The program SHALL name the service in each log line that the service writes.
* The program SHALL stop when one of its services stops.

## Known Limitations

* The peer does not accept postings from other peers.
* Some log lines do not name a service, for example the lines that a library writes.
