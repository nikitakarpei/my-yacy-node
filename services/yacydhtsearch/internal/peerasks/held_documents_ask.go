// Package peerasks holds the asks this node puts to peers and what one peer
// answered for each — the documents it matched, the documents it holds for one
// word together with the documents it matched for that word, or the metadata it
// holds for the documents the ask names. A matched document carries the count
// of a word the ask named without the word itself, zero when the peer counted
// none, so only the spread that named the words of the ask can say which word
// the count belongs to.
package peerasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
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
	DocumentsHeldForTheWord         []yacymodel.URLHash
	MatchedDocuments                []MatchedDocument
	AmountOfDocumentsHeldForTheWord yacymodel.Optional[int]
}
