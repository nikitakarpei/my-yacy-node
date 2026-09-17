package peerasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type MatchedDocumentsAsk struct {
	Peer          peerdirectory.AskablePeer
	Partition     uint
	WordsToMatch  []yacymodel.Hash
	ExcludedWords []yacymodel.Hash
	Language      string
	ItemsCeiling  int
}

type AnsweredMatchedDocumentsAsk struct {
	Ask              MatchedDocumentsAsk
	MatchedDocuments []MatchedDocument
}
