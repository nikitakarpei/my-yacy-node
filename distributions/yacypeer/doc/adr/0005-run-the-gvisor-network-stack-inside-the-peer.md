# 5. Run the gVisor network stack inside the peer

Status: accepted

## Context

The Yggdrasil node of the peer gives raw IPv6 packets. The peer serves and sends HTTP, so it
needs TCP over these packets. Without a TUN device, the kernel cannot supply TCP.

`gvisor.dev/gvisor` is the gVisor sandbox as a Go module, under the Apache 2.0 licence. Its
network stack has IPv6 and TCP in Go, without cgo. Yggstack, Tailscale, and WireGuard use it.

Smaller stacks are available. `github.com/soypat/lneto` has IPv6 and TCP, but it is new.
lwIP needs cgo. QUIC on the Yggdrasil node does not let nodes with a TUN device reach the peer.

## Decision

Link `gvisor.dev/gvisor` into `yacypeer`, and use its network stack only for the Yggdrasil
address of the peer.

## Consequences

* Peers on the Yggdrasil network reach the peer with plain HTTP over TCP.
* The binary is larger, and each TCP connection over Yggdrasil has its own buffers.
* A different stack can replace gVisor in `internal/ipv6stack` only.
