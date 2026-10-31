package yacymodel_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestPostingsPerURLGroupThePostingsOfEachURLAndDropRepeats(t *testing.T) {
	random := seededRandom(24)
	firstPage := randomPagePostings(random, 3)
	secondPage := randomPagePostings(random, 2)
	repeated := firstPage[1]
	repeated.Hits++

	perURL, repeatedPostings := yacymodel.PostingsPerURLFrom([]yacymodel.RWIPosting{
		firstPage[0], secondPage[0], firstPage[1], repeated, firstPage[2], secondPage[1],
	})

	if len(perURL) != 2 ||
		!slices.Equal(perURL[0], firstPage) ||
		!slices.Equal(perURL[1], secondPage) {
		t.Errorf("postings per url = %+v, want the first page, then the second", perURL)
	}
	if !slices.Equal(repeatedPostings, []yacymodel.RWIPosting{repeated}) {
		t.Errorf("repeated postings = %+v, want %+v", repeatedPostings, repeated)
	}
}
