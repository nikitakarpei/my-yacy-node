# 7. Run a Yggdrasil node inside the peer

Status: accepted

## Context

Yggdrasil is an overlay network. Each node has an IPv6 address that comes from its key, and
other nodes can reach that address through NAT. A peer behind NAT can thus serve other peers
without a port forward.

Without a built-in node, an operator of `yacypeer` must run the Yggdrasil daemon. The daemon
needs a TUN device and the `CAP_NET_ADMIN` capability.

`github.com/yggdrasil-network/yggdrasil-go` is Yggdrasil as a Go module, under the LGPLv3
licence with an exception for static linking.

## Decision

Link `github.com/yggdrasil-network/yggdrasil-go` into `yacypeer`. Start the node only when the
operator names Yggdrasil peers or Yggdrasil listen URIs.

## Consequences

* The peer can join the Yggdrasil network without a TUN device and without root.
* The node gives raw IPv6 packets, so the peer needs a TCP/IP stack in the process.
* The binary is larger.
