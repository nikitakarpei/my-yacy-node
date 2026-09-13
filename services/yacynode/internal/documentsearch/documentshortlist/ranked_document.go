package documentshortlist

import (
	"cmp"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchrelevance"
)

type RankedDocument struct {
	JoinedPosting yacymodel.RWIPosting
	Relevance     searchrelevance.Relevance
	TermSpread    int
}

func compareInRelevanceOrder(a, b RankedDocument) int {
	return cmp.Or(
		cmp.Compare(b.Relevance, a.Relevance),
		cmp.Compare(a.TermSpread, b.TermSpread),
		yacymodel.CompareInAlphabetOrder(
			a.JoinedPosting.URLHash.String(),
			b.JoinedPosting.URLHash.String(),
		),
	)
}
