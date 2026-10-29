# Secure peer protocol

The node serves the YaCy peer protocol over HTTP and over TLS on one port, the
port of `YACY_PEER_ADDR`. The certificate of the node proves that the node holds
its peer key. The seed of the node announces this port as its port and as its
secure port.

The metric `yacynode_peerport_connections_total` counts the connections to the
port by scheme, `http` or `https`.

## Calls to other peers

When the signed seed of a peer announces a secure port, the node calls that
peer only over TLS. The node accepts only a certificate that holds the identity
of the peer and that its key endorses. The node does not use plain HTTP for
that peer.

When the signed seed of a peer announces no secure port, the node calls that
peer only over plain HTTP.

When a peer has no signed seed, the node tries HTTPS first for each DNS name of
the peer, with a certificate from a public authority, and then plain HTTP.

## Java peers

Java peers use HTTP by default. A Java peer uses TLS only when its operator
turns on `network.unit.protocol.https.preferred` or
`remotesearch.https.preferred`.

## Network

Other peers must reach the port. TLS must end in the node, so do not put a
reverse proxy in front of the port.
