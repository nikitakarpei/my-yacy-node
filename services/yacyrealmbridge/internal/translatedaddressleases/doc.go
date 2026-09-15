// Package translatedaddressleases owns the one address a peer hash is
// translated to in a realm: taken once from that realm's translated address
// space, renewed while the bridge holds the peer, and free again when the lease
// expires. It also gives back the hash behind a translated address, and leases
// nothing once the space is full.
package translatedaddressleases
