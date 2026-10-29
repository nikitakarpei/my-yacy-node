# 5. Bind TLS keys to the peer identity

Date: 2026-10-29

## Status

Proposed. Not built. The open options below are decided before it is built.

## Context

Peers talk plain HTTP. A party on the path can read the requests and change the answers. Most
peers have only an IP address, so a certificate from a public authority is not available.

The peer hash commits to the operational key (ADR 0001), so the operational key can vouch for a
TLS key.

## Decision

At each start, a peer makes a new ECDSA P-256 TLS key and a self-signed certificate for it. The
certificate holds a non-critical extension with the OID
`2.25.62252900205088142946132973079864517537`. Its value is the DER form of:

```text
SEQUENCE { identity UTF8String, signature OCTET STRING }
```

The signature is Ed25519 by the operational key over:

```text
"yacy-tls-key/1" ‖ length-prefixed(network name, peer hash, DER public key of the TLS key)
```

- The TLS key is not the operational key, so the key that signs seeds stays out of the TLS stack.
- P-256 works with all Java versions that run YaCy. Ed25519 in TLS needs Java 15 or later.
- The peer serves TLS 1.3 on its own port. The seed announces this port with the SSL flag in
  `Flags` and with `PortSSL`. The address signature (ADR 0004) covers both.

Rules for a client that connects to a peer hash:

- The identity in the extension must give that peer hash. If the client has a bound identity
  for the peer, the two identities must be equal.
- The signature must be valid for the TLS key of the connection.
- The client ignores the names and the dates of the certificate.
- A bound peer whose signed seed announces a secure port gets TLS only. The client does not use
  plain HTTP for this peer.
- For other peers, the client keeps the current rules.

## Open options

- Client certificates. A peer can send its own certificate, so that the server knows the peer
  hash of the caller. Java peers do not send one, so the server must accept connections without
  one. What the server does with a known caller is not decided.

## Consequences

Traffic between two new peers is private, and a party on the path cannot change it. The IP
addresses, the ports, the sizes and the times stay visible.

Java peers use `PortSSL` only when their operator sets `network.unit.protocol.https.preferred`
or `remotesearch.https.preferred`. Both are off by default. Java accepts any certificate. When
TLS fails, Java clears the SSL flag in its copy of the seed and uses plain HTTP. Traffic with
Java peers stays plain HTTP by default.

TLS must end in the peer. A reverse proxy in front of the TLS port breaks the check.

The operator forwards one more port.

TLS protects one connection only. Data that peers pass on, such as seeds, needs its own
signature.

With delegation (ADR 0003), the client checks the signature with the delegated operational key.
