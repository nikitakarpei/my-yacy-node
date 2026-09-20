package queryanswers_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestTheLinkCountsOfAPostingCountTheLinksThePostingReported(t *testing.T) {
	t.Parallel()

	linkCounts := queryanswers.LinkCountsFrom(yacymodel.RWIPosting{
		LocalLinks:    12,
		ExternalLinks: 7,
	})

	if linkCounts.LocalLinks != 12 || linkCounts.ExternalLinks != 7 {
		t.Fatalf("the link counts read %+v, want 12 local and 7 external", linkCounts)
	}
}

func TestAFoundDocumentOfMetadataAloneHoldsNoLinkCounts(t *testing.T) {
	t.Parallel()

	foundDocument := queryanswers.FoundDocumentFrom(yacymodel.URLMetadata{
		Address:       "https://example.org/weather",
		LocalLinks:    12,
		ExternalLinks: 7,
	})

	if foundDocument.LinkCounts.Present() {
		t.Fatal("the found document holds link counts, want none until a posting reports them")
	}
}
