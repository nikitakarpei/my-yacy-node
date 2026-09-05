// Package indexabstract builds the index abstracts a search request asked for.
// One index abstract names a term and the documents this node holds for it. A
// peer reads index abstracts to plan which peers to ask next, so they carry
// document hashes only, never metadata.
package indexabstract

import (
	"cmp"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termpostings"
)

type IndexAbstracts map[yacymodel.Hash][]yacymodel.URLHash

func IndexAbstractsFor(
	requested RequestedIndexAbstracts,
	criteria searchcriteria.Criteria,
	matchesForQueryTerms map[yacymodel.Hash]termpostings.Match,
	matchesForIndexAbstractTerms map[yacymodel.Hash]termpostings.Match,
) IndexAbstracts {
	switch requested := requested.(type) {
	case IndexAbstractOfTermWithMostPostings:
		return indexAbstractOfTermWithMostPostings(criteria, matchesForQueryTerms)
	case IndexAbstractsOfTerms:
		return indexAbstractsOfTerms(requested.Terms, matchesForIndexAbstractTerms)
	default:
		return nil
	}
}

func indexAbstractOfTermWithMostPostings(
	criteria searchcriteria.Criteria,
	matchesForQueryTerms map[yacymodel.Hash]termpostings.Match,
) IndexAbstracts {
	if len(criteria.Terms) <= 1 || len(criteria.RequiredDocuments) != 0 {
		return nil
	}
	term, ok := termWithMostPostingsOf(matchesForQueryTerms)
	if !ok {
		return nil
	}

	return IndexAbstracts{
		term: documentHashesOf(matchesForQueryTerms[term].PostingPerDocument),
	}
}

func termWithMostPostingsOf(
	matches map[yacymodel.Hash]termpostings.Match,
) (yacymodel.Hash, bool) {
	var (
		termWithMostPostings yacymodel.Hash
		mostPostings         int
		found                bool
	)
	for term, match := range matches {
		if !found || match.PostingsHeld > mostPostings ||
			match.PostingsHeld == mostPostings &&
				cmp.Compare(term.String(), termWithMostPostings.String()) < 0 {
			termWithMostPostings = term
			mostPostings = match.PostingsHeld
			found = true
		}
	}

	return termWithMostPostings, found
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

func indexAbstractsOfTerms(
	terms []yacymodel.Hash,
	matchesForIndexAbstractTerms map[yacymodel.Hash]termpostings.Match,
) IndexAbstracts {
	abstracts := make(IndexAbstracts, len(terms))
	for _, term := range terms {
		abstracts[term] = documentHashesOf(
			matchesForIndexAbstractTerms[term].PostingPerDocument,
		)
	}

	return abstracts
}
