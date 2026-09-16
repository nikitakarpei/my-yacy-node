package peerpresence

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerAtAddress struct {
	Hash    yacymodel.Hash
	Address string
}

type ObservedPeer struct {
	PeerAtAddress
	FirstAnsweredAt  time.Time
	LatestAnsweredAt time.Time
	Presence         time.Duration
}

type PeerAnswered struct {
	PeerAtAddress
	AnsweredAt time.Time
}
