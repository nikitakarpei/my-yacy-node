# YaCy-Compatible RWI Node — Technical Specification

## Context

YaCy DHT RWI participation assumes globally reachable senior peers running the full Java
implementation, which is heavy for low-resource hardware.
The project provides a lightweight Go implementation of a senior YaCy node focused on DHT RWI
storage and serving, suitable for low-resource Linux-class devices.
Additional YaCy-related capabilities can be provided by separate services when needed.

## Non-Goals

* Built-in web crawling.
* Built-in HTML parsing, proxying, or content fetching.
* Full-text indexing or local search UI.
* Solr, Lucene, Elasticsearch, or equivalent full-text integration.
* Citation graphs, webgraph indexing, or media indexing.
* A full-scale Go YaCy node beyond DHT RWI storage and serving.

## Functional Requirements

* The node SHALL advertise one YaCy Senior peer identity.
* The node SHALL derive its peer hash from the peer key that the operator states, and SHALL refuse
  to start without a valid key or when its data holds the hash of a different key.
* The node SHALL sign the addresses in its seed with its peer key.
* The node SHALL serve the peer protocol over TLS that its peer key proves.
* The node SHALL drop each seed of a peer that breaks the identity bound from its first signed seed.
* The node SHALL allow operators to configure the peer name it advertises.
* The node SHALL allow operators to configure the public hosts and port advertised in its YaCy seed.
* The node SHALL advertise the public IP address that most peers reached by IP address saw it
  connect from, when the operator does not configure the host.
* The node SHALL announce itself through configured YaCy seedlists.
* The node SHALL require operators to configure a proxy for outbound connections.
* The node SHALL be reachable through one stable public endpoint.
* The node SHALL contact a peer as [Secure peer protocol](secure-peer-protocol.md) states.
* The node SHALL announce in peer-liveness responses only its own seed and peers obtained from
  configured seedlists, and SHALL NOT redistribute peers self-reported in inbound requests.
* The node SHALL honor the requested peer count in peer-liveness requests and select the announced
  peers at random.
* The node SHALL receive inbound DHT RWI postings.
* The node SHALL receive URL metadata associated with RWI postings.
* The node SHALL serve remote RWI search requests.
* The node SHALL serve the stored URL metadata of the URL hashes a peer names.
* The node SHALL answer RWI capacity and status queries.
* The node SHALL reject remote crawl work.
* The node SHALL let operators give the messages that peers send to a webhook, and SHALL refuse
  messages when no webhook is set.
* The node SHALL send a message to a webhook at a public address only over HTTPS through the
  outbound proxy.
* The node SHALL store accepted RWI postings and the URL metadata those postings reference.
* The node SHALL NOT store a posting whose word is a stopword of the posting language.
* The node SHALL allow operators to configure its storage quota.
* The node SHALL NOT store full document bodies.
* The node SHALL let operators enable outbound DHT distribution, which is off by default.
* The node SHALL let operators configure how many responsible peers hold each posting.
* The node SHALL count itself as one of those peers when the DHT makes it responsible for a
  posting.
* The node SHALL let operators configure the fewest known reachable peers distribution needs,
  and SHALL pause distribution while the network is below it.
* When enabled, the node SHALL offer each stored posting to the peers the DHT makes
  responsible for it.
* The node SHALL treat a posting as distributed only once that many responsible peers hold it.
* When enabled, the node SHALL delete the postings of one word in one ring partition once that
  many responsible peers closer to their DHT position than the node hold all of them.

## Non-Functional Requirements

* The node SHALL durably retain accepted records before acknowledging them.
* The node SHALL apply backpressure when it cannot durably retain further accepted records.
* The node SHALL keep memory usage bounded independently of total stored RWI size.
* The node SHALL set explicit limits on all caches, queues, buffers, batches, and request bodies.
* The node SHALL complete requests within bounded deadlines.
* The node SHALL prefer availability and data integrity over intake throughput.
* The node SHALL serve and store first and distribute only with the resources left over,
  within operator-set bounds.
* The node SHALL support low-resource Linux-class devices, including Raspberry-Pi-class hardware.
* The node SHALL preserve compatibility with standard YaCy peer-to-peer contracts.
* The node SHALL NOT require rebuilding the complete index in memory after restart.
* The node SHALL NOT corrupt persistent state when disk is exhausted.
* Storage engines SHALL be replaceable behind a narrow interface.
* Operational behavior SHALL be observable through machine-readable metrics.
