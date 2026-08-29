# 1. Join the YaCy network with one peer

> "Can a spare box support the YaCy network?"

A peer uses spare storage and bandwidth to keep YaCy's shared reverse word
index available. Other peers can store postings on it and query them. You can
support the network without running a crawler.

## What this chapter adds

- `yacy-rwi-node` stores part of YaCy's shared index and answers searches from
  other peers.
- `smokescreen` is the egress proxy for every outbound connection the node
  opens. It rejects private destinations, so addresses supplied by the network
  cannot reach private services through the node.

## Start

Allow inbound connections to port 8090, then start the stack:

```sh
docker compose up -d
```

## Use

Check the peer roster metrics after the first network announcement:

```sh
curl -fsS localhost:9090/metrics | grep '^peerroster_.*_peers '
docker compose logs -f yacy-rwi-node
```

A nonzero reachable peer count confirms that the node discovered and contacted
other peers.

## More information

- [Node configuration](../../../../services/yacynode/doc/configuration.md)
- [Node behavior](../../../../services/yacynode/doc/specification.md)
