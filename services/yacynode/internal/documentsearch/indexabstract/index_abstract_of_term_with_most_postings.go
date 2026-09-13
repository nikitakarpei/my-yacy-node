package indexabstract

import (
	"cmp"
	"maps"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termpostings"
)

type IndexAbstractOfTermWithMostPostings struct{}

func (IndexAbstractOfTermWithMostPostings) indexAbstracts(
	matchesForQueryTerms map[yacymodel.Hash]termpostings.Match,
	_ map[yacymodel.Hash]termpostings.Match,
) IndexAbstracts {
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
	terms := termsWithDocumentsOf(matches)
	if len(terms) == 0 {
		return yacymodel.Hash{}, false
	}

	return slices.MinFunc(terms, func(a, b yacymodel.Hash) int {
		return cmp.Or(
			cmp.Compare(matches[b].PostingsHeld, matches[a].PostingsHeld),
			cmp.Compare(a.String(), b.String()),
		)
	}), true
}

func termsWithDocumentsOf(matches map[yacymodel.Hash]termpostings.Match) []yacymodel.Hash {
	return slices.DeleteFunc(
		slices.Collect(maps.Keys(matches)),
		func(term yacymodel.Hash) bool { return len(matches[term].PostingPerDocument) == 0 },
	)
}
