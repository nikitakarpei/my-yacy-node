package yacycrawlcontract_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestCrawledPageStreamNameIsRepresentationQualified(t *testing.T) {
	for representation, want := range map[yacycrawlcontract.PageRepresentationKind]string{
		yacycrawlcontract.PageRepresentationKindRWI:      "YACY_CRAWL_PAGE_RWI",
		yacycrawlcontract.PageRepresentationKindText:     "YACY_CRAWL_PAGE_TEXT",
		yacycrawlcontract.PageRepresentationKindMarkdown: "YACY_CRAWL_PAGE_MARKDOWN",
	} {
		if got := yacycrawlcontract.CrawledPageStreamName(representation); got != want {
			t.Errorf("CrawledPageStreamName(%q) = %q, want %q", representation, got, want)
		}
	}
}

func TestCrawledPageSubjectIsRepresentationQualified(t *testing.T) {
	for representation, want := range map[yacycrawlcontract.PageRepresentationKind]string{
		yacycrawlcontract.PageRepresentationKindRWI:      "yacy.crawl.page.rwi",
		yacycrawlcontract.PageRepresentationKindText:     "yacy.crawl.page.text",
		yacycrawlcontract.PageRepresentationKindMarkdown: "yacy.crawl.page.markdown",
	} {
		if got := yacycrawlcontract.CrawledPageSubject(representation); got != want {
			t.Errorf("CrawledPageSubject(%q) = %q, want %q", representation, got, want)
		}
	}
}
