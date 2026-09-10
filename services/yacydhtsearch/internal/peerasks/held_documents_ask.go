// Package peerasks holds the asks this node puts to peers and what one peer
// answered for each — the items it matched, the documents it holds for one
// word together with the items it reports for that word, or the metadata it
// holds for the documents the ask names.
package peerasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type HeldDocumentsAsk struct {
	Peer          peerdirectory.AskablePeer
	Word          yacymodel.Hash
	ExcludedWords []yacymodel.Hash
	Language      string
	ItemsCeiling  int
}

type AnsweredHeldDocumentsAsk struct {
	Ask                             HeldDocumentsAsk
	Documents                       []yacymodel.URLHash
	Items                           []searchresult.Item
	AmountOfItemsWithAPosting       int
	AmountOfDocumentsHeldForTheWord int
}
