package peerasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type URLMetadataAsk struct {
	Peer      peerdirectory.AskablePeer
	Documents []yacymodel.URLHash
}

type AnsweredURLMetadataAsk struct {
	Ask                    URLMetadataAsk
	MetadataOfEachDocument []yacymodel.URLMetadata
}
