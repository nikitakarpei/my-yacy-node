package peerroster

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
)

type PeerRecord struct {
	Callee          peerwire.Callee
	Addresses       yacymodel.PeerAddresses
	AnsweredBaseURL yacymodel.Optional[peerwire.BaseURL]
}

func (r PeerRecord) BaseURLs() []peerwire.BaseURL {
	return r.Callee.BaseURLsFrom(r.Addresses)
}
