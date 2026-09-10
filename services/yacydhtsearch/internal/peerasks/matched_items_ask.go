package peerasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type MatchedItemsAsk struct {
	Peer          peerdirectory.AskablePeer
	WordsToMatch  []yacymodel.Hash
	ExcludedWords []yacymodel.Hash
	Language      string
	ItemsCeiling  int
}

type AnsweredMatchedItemsAsk struct {
	Ask   MatchedItemsAsk
	Items []searchresult.Item
}
