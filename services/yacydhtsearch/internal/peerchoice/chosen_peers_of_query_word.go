package peerchoice

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ChosenPeersOfQueryWord struct {
	QueryWord   yacymodel.Hash
	ChosenPeers []ChosenPeer
}
