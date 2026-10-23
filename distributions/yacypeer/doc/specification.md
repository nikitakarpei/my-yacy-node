# yacypeer — Technical Specification

## Context

A YaCy peer of this repository is a set of services that share a NATS server. `yacypeer`
runs these services and their NATS server as one program, connected and ready to use.

## Non-Goals

* Serving a search page; SearXNG stays a separate program.
* Showing metrics; Prometheus and Grafana stay separate programs.
* Running a part of the peer alone.

## Functional Requirements

* The program SHALL run `yacynode`, `yacydhtsearch`, `corpustext`, and `corpusmarkdown`,
  and a NATS server that only the local host can reach.
* The program SHALL keep its local data in one data directory.
* The program SHALL set the settings that connect the services, and SHALL NOT start when
  the operator sets one of them or an ops setting of a service.
* The program SHALL give a setting with a service prefix priority for that service.
* The program SHALL serve the metrics of the process and of each service on one address.
* The program SHALL name the service in each log line that the service writes.
* The program SHALL stop when one of its services stops.

## Known Limitations

* No service writes page offers yet, so `yacynode`, `corpustext`, and `corpusmarkdown` log
  errors while they wait for them.
* The peer does not accept postings from other peers.
* Some log lines do not name a service, for example the lines that a library writes.
