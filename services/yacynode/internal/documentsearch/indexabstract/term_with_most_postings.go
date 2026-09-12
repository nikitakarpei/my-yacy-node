package indexabstract

import (
	"cmp"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termpostings"
)

func indexAbstractOfTermWithMostPostings(
	matchesForQueryTerms map[yacymodel.Hash]termpostings.Match,
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
	var (
		termWithMostPostings yacymodel.Hash
		mostPostings         int
		found                bool
	)
	for term, match := range matches {
		if len(match.PostingPerDocument) == 0 {
			continue
		}
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
