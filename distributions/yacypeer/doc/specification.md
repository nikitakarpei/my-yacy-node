# yacypeer — Technical Specification

## Context

A YaCy peer of this repository is a set of services and a NATS server. An operator with a
small workload must run, configure, and connect each one. `yacypeer` is one binary that
runs the services of a peer and the NATS server they share, connected and ready to use.

## Non-Goals

* Serving a search page; SearXNG stays a separate program.
* Running a service of the peer without the others.
* Running more than one instance on the same data directory.
* Collecting or showing metrics; Prometheus and Grafana stay separate programs.

## Functional Requirements

* The binary SHALL run `yacynode`, `yacydhtsearch`, `corpustext`, and `corpusmarkdown` in
  one process.
* The binary SHALL run a NATS JetStream server that only the loopback interface reaches,
  and SHALL connect every member that uses NATS to it.
* The binary SHALL keep the data of each member, and of the NATS server, in its own
  directory under the data directory.
* The binary SHALL give each member the value it wires, else the setting with the member
  name as a prefix, else the setting without a prefix.
* The binary SHALL NOT start when the operator sets a setting it wires, and SHALL name that
  setting.
* The binary SHALL serve the runtime metrics of the process at `/metrics`, and the metrics
  of each member at `/metrics/<member>`, on one address.
* The binary SHALL NOT start when the operator sets an ops setting of a member, and SHALL
  name that setting.
* The binary SHALL stop every member when one member stops.
* The binary SHALL name the member in the `member` field of each log line that the member
  writes from its run.

## Non-Functional Requirements

* A member SHALL behave as the same service does when it runs alone, except for the
  settings the binary wires.

## Known Limitations

* All members write to one log. A line does not name a member when a library writes it
  through its own logger, or when a member writes it while it answers a request.
* No member writes page offers yet, so `yacynode`, `corpustext`, and `corpusmarkdown` keep
  trying to open them.
