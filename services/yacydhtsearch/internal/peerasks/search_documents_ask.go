// Package peerasks holds the asks the service puts to peers and what one peer
// answered for each.
package peerasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type SearchDocumentsAsk struct {
	Peer                    peerdirectory.AskablePeer
	Word                    yacymodel.Hash
	ExcludedWords           []yacymodel.Hash
	Language                string
	DocumentsToMatch        []yacymodel.URLHash
	Abstract                bool
	MatchedDocumentsCeiling yacymodel.Optional[int]
}

type SearchDocumentsAskOutcome = AskOutcome[SearchDocumentsAsk, AnsweredSearchDocumentsAsk]

type SearchDocumentsAskOutcomes = AskOutcomes[SearchDocumentsAsk, AnsweredSearchDocumentsAsk]

type AnsweredSearchDocumentsAsk struct {
	Ask                             SearchDocumentsAsk
	Abstract                        []yacymodel.URLHash
	MatchedDocuments                []MatchedDocument
	AmountOfDocumentsHeldForTheWord yacymodel.Optional[int]
	PeerSearched                    bool
}
