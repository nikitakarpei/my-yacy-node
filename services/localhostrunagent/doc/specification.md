# localhostrunagent — Technical Specification

## Context

A YaCy peer must give other peers an address that reaches its port. An operator
behind NAT, carrier-grade NAT, or a changing address has no such address, and
the node cannot announce itself. `localhostrunagent` is a sidecar of one node
that holds a localhost.run reverse tunnel, leases the public hostname the
provider assigns to the node process, and forwards tunnel traffic to the node.
The node then announces a hostname that reaches it, with no router rule and no
public IP.

## Non-Goals

* Serving the peer protocol, storing postings, or answering searches.
* Terminating TLS or presenting a certificate for the assigned hostname.
* Keeping one hostname across tunnels, which the provider decides.
* Retrying a tunnel in place, or supervising its own restart.
* Serving more than one node.
* Choosing the tunnel provider at run time.

## Functional Requirements

* The agent SHALL hold one tunnel and SHALL report the end of that tunnel as a
  failure.
* The agent SHALL take the public hostname from the provider's forward event and
  SHALL reject an assigned address that is not a public hostname.
* The agent SHALL treat a second forward event that names a different hostname as
  a failure, and SHALL accept one that repeats the assigned hostname.
* The agent SHALL grant `YACY_ADVERTISE_HOST` and `YACY_ADVERTISE_PORT` to the
  node process over the lease socket after the tunnel opens.
* The agent SHALL stop when the tunnel, the ingress, or the lease consumer stops.
* The agent SHALL accept the host key on the first connection, SHALL persist it,
  and SHALL stop when a later host key differs.
* The ingress SHALL require a PROXY protocol header on every connection and SHALL
  reject a connection that starts without one.
* The ingress SHALL replace every forwarding header of a request with the one
  address the PROXY protocol header carries, so a caller cannot forge a client
  address.
* The ingress SHALL send every request to the node origin from one bound local
  address, so the node can trust that address alone as a proxy.

## Non-Functional Requirements

* The agent SHALL bound the provider event line it reads and the request header
  it accepts, so neither can exhaust memory.
* The agent SHALL hold the exit of a process that failed within the shortest
  interval between two starts, so an outer supervisor cannot restart it sooner.
* The agent SHALL exit at once when it ran longer than that interval, and when it
  receives a signal.
* The agent SHALL write only to its state directory and a temporary directory,
  so a deployment can give it a read-only root and no privileges of its own.

## Known Limitations

* The callback that promotes the node to `senior` must reach it through the
  tunnel. A slow tunnel, or one whose hostname changed, leaves the node
  `junior`.
* A hostname change ends the lease and restarts the node, which loses the
  uptime the node reports to other peers.
* Tunnel traffic reaches the node as plain HTTP, so the provider sees it.
