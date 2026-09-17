// Package peerdirectory owns the peers this service knows and which address
// each answers on. It keeps the address a peer has answered on even after the
// peer goes silent, so a seedlist that no longer names that address cannot
// take it away from the peer. A full directory keeps a peer it holds over an
// offered peer of the same standing, and lends a bounded share of itself in
// every admission to peers it does not hold, so it goes on finding peers it
// has never met.
package peerdirectory

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type KnownPeer struct {
	Hash            yacymodel.Hash
	Addresses       []string
	AnsweredAddress string
	AdmittedAt      time.Time
	AnsweredAt      time.Time
	WentSilentAt    time.Time
}

func (peer KnownPeer) answersNow() bool {
	return peer.AnsweredAddress != "" && peer.AnsweredAt.After(peer.WentSilentAt)
}

type CandidatePeer struct {
	Hash      yacymodel.Hash
	Addresses []string
}

type AskablePeer struct {
	Hash    yacymodel.Hash
	Address string
}
