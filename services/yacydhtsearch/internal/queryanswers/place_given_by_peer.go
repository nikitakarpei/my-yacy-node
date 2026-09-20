package queryanswers

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PlaceGivenByPeer struct {
	Peer                 yacymodel.Hash
	ReliabilityOfThePeer float64
	Place                int
}
