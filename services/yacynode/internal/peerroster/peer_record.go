package peerroster

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
)

type PeerRecord struct {
	Addresses       yacymodel.PeerAddresses
	Identity        yacymodel.Optional[yacymodel.PeerIdentity]
	AnsweredBaseURL yacymodel.Optional[peerwire.BaseURL]
}
