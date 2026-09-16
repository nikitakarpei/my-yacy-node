package peeranswerhistory

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerAtAddress struct {
	Hash    yacymodel.Hash
	Address string
}

type PeerAnswer struct {
	PeerAtAddress
	AnsweredAt time.Time
}
