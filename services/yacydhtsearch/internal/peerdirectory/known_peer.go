// Package peerdirectory owns the peers this service knows, which address each
// answers on, and when each may next be asked. It keeps the address a peer has
// answered on even after the peer goes silent, so a seedlist that no longer
// names that address cannot take it away from the peer.
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
	ChosenAt        time.Time
	AnsweredAt      time.Time
	WentSilentAt    time.Time
}

func (peer KnownPeer) answersNow() bool {
	return peer.AnsweredAddress != "" && peer.AnsweredAt.After(peer.WentSilentAt)
}

type AskablePeer struct {
	Hash    yacymodel.Hash
	Address string
}
