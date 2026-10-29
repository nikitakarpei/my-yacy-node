# Secure peer protocol

The node serves the YaCy peer protocol over TLS on a second port, the secure
port. The certificate of the node proves that the node holds its peer key. The
seed of the node announces the secure port.

| Variable | Default | Description |
| --- | --- | --- |
| `YACY_SECURE_PEER_ENABLED` | `true` | Serve and announce the secure port. Set it to `false` when other peers cannot reach a second port of the node. |
| `YACY_SECURE_PEER_ADDR` | `:8443` | Listen address for the secure peer protocol. |
| `YACY_ADVERTISE_SECURE_PORT` | _(the `YACY_SECURE_PEER_ADDR` port)_ | Secure port other peers use to reach you. |

## Calls to other peers

When the signed seed of a peer announces a secure port, the node calls that
peer only over TLS. The node accepts only a certificate that the key of that
peer hash endorses. The node does not use plain HTTP for that peer.

The node calls all other peers as before.

## Java peers

Java peers use the plain port by default. A Java peer uses the secure port only
when its operator turns on `network.unit.protocol.https.preferred` or
`remotesearch.https.preferred`.

## Network

Other peers must reach the secure port. TLS must end in the node, so do not put
a reverse proxy in front of the secure port.
