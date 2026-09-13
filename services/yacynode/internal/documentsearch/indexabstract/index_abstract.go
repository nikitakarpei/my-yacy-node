// Package indexabstract builds the index abstracts a search request asked for.
// One index abstract names a term and the documents this node holds for it. A
// peer reads index abstracts to plan which peers to ask next, so they carry
// document hashes only, never metadata.
package indexabstract

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termpostings"
)

type IndexAbstracts map[yacymodel.Hash][]yacymodel.URLHash

func IndexAbstractsFor(
	requested RequestedIndexAbstracts,
	matchesForQueryTerms map[yacymodel.Hash]termpostings.Match,
	matchesForIndexAbstractTerms map[yacymodel.Hash]termpostings.Match,
) IndexAbstracts {
	abstracts := make(IndexAbstracts, len(requested))
	for _, requestedAbstract := range requested {
		maps.Copy(
			abstracts,
			requestedAbstract.indexAbstracts(matchesForQueryTerms, matchesForIndexAbstractTerms),
		)
	}

	return abstracts
}

func documentHashesOf(
	postingPerDocument map[yacymodel.URLHash]yacymodel.RWIPosting,
) []yacymodel.URLHash {
	hashes := make([]yacymodel.URLHash, 0, len(postingPerDocument))
	for documentHash := range postingPerDocument {
		hashes = append(hashes, documentHash)
	}

	return hashes
}
