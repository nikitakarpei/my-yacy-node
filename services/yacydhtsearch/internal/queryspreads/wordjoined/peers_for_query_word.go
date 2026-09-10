package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type peersForQueryWord struct {
	word  yacymodel.Hash
	peers []peerdirectory.AskablePeer
}
