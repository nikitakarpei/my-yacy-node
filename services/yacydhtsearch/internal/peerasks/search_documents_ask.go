// Package peerasks holds the asks the service puts to peers and what one peer
// answered for each.
package peerasks

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type SearchDocumentsAsk struct {
	Peer             peerdirectory.AskablePeer
	Partition        uint
	Word             yacymodel.Hash
	ExcludedWords    []yacymodel.Hash
	Language         string
	DocumentsToMatch []yacymodel.URLHash
	ItemsCeiling     int
}

type SearchDocumentsAskOutcome = AskOutcome[SearchDocumentsAsk, AnsweredSearchDocumentsAsk]

type SearchDocumentsAskOutcomes = AskOutcomes[SearchDocumentsAsk, AnsweredSearchDocumentsAsk]

type AnsweredSearchDocumentsAsk struct {
	Ask                             SearchDocumentsAsk
	Abstract                        []yacymodel.URLHash
	MatchedDocuments                []MatchedDocument
	AmountOfDocumentsHeldForTheWord yacymodel.Optional[int]
}

func (answeredAsk AnsweredSearchDocumentsAsk) IgnoredTheDocumentsToMatch() bool {
	if len(answeredAsk.Ask.DocumentsToMatch) == 0 {
		return false
	}
	for _, document := range answeredAsk.Abstract {
		if !slices.Contains(answeredAsk.Ask.DocumentsToMatch, document) {
			return true
		}
	}

	return false
}
