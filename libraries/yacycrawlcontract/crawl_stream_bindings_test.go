package yacycrawlcontract_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestCrawledPageStreamNameIsRepresentationQualified(t *testing.T) {
	for representation, want := range map[yacycrawlcontract.PageRepresentationKind]string{
		yacycrawlcontract.PageRepresentationKindRWI: "YACY_CRAWL_PAGE_RWI",
	} {
		if got := yacycrawlcontract.CrawledPageStreamName(representation); got != want {
			t.Errorf("CrawledPageStreamName(%q) = %q, want %q", representation, got, want)
		}
	}
}

func TestCrawledPageSubjectIsRepresentationQualified(t *testing.T) {
	for representation, want := range map[yacycrawlcontract.PageRepresentationKind]string{
		yacycrawlcontract.PageRepresentationKindRWI: "yacy.crawl.page.rwi",
	} {
		if got := yacycrawlcontract.CrawledPageSubject(representation); got != want {
			t.Errorf("CrawledPageSubject(%q) = %q, want %q", representation, got, want)
		}
	}
}
