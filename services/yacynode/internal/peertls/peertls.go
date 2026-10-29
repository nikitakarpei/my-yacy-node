// Package peertls carries the peer key into TLS. It makes the certificate that this peer
// serves on its secure port, and it lets a call go to a peer only when the peer holds the
// key of the peer hash that the call names.
package peertls

const endorsementScheme = "yacy-tls-key"

const endorsementPartSeparator = ":"
