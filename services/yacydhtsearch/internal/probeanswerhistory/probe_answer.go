package probeanswerhistory

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerAtAddress struct {
	Hash    yacymodel.Hash
	Address string
}

type ProbeAnswer struct {
	PeerAtAddress
	AnsweredAt time.Time
}
