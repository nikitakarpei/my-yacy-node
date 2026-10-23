# 2. Run the egress proxy inside the peer

Status: accepted

## Context

`yacynode`, `yacydhtsearch`, `yacycrawler`, and `pagescrape` send their outbound requests
through an egress proxy. The proxy refuses requests to loopback and private addresses.
Without a built-in proxy, an operator of `yacypeer` must also run and connect smokescreen.

`github.com/stripe/smokescreen` is smokescreen as a Go module, under the MIT licence. The
repository already builds its image from the same commit.

## Decision

Link `github.com/stripe/smokescreen` into `yacypeer`, and run it on the local host only,
unless the operator names an egress proxy.

## Consequences

* The peer needs no egress proxy outside it, but can use one that the operator names.
* Nothing outside the host can send requests through the built-in proxy.
* The binary is larger.
