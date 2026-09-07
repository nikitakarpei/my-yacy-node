// Package peerdirectory owns the peers this service knows, which address each
// answers on, and when each may next be asked.
package peerdirectory

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type KnownPeer struct {
	Hash             yacymodel.Hash
	Addresses        []string
	AnsweringAddress string
	AdmittedAt       time.Time
	AskedAt          time.Time
	AnsweredAt       time.Time
}
type AskablePeer struct {
	Hash    yacymodel.Hash
	Address string
}
