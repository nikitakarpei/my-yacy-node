package contentformatgraph

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

func TestValidateAcceptsReachableFormat(t *testing.T) {
	graph := New([]crawlcapability.PageDerivation{
		scriptedDerivation{
			source: crawlcapability.PageContentFormatDocumentHTML,
			target: crawlcapability.PageContentFormatReadableHTML,
		},
		scriptedDerivation{
			source: crawlcapability.PageContentFormatReadableHTML,
			target: crawlcapability.PageContentFormatMarkdown,
		},
	})
	if err := graph.Validate([]crawlcapability.PageContentFormat{
		crawlcapability.PageContentFormatMarkdown,
	}); err != nil {
		t.Fatalf("markdown is reachable from document-html: %v", err)
	}
}

func TestValidateRejectsUnproducedFormat(t *testing.T) {
	graph := New(nil)
	if err := graph.Validate([]crawlcapability.PageContentFormat{
		crawlcapability.PageContentFormatReadableText,
	}); err == nil {
		t.Fatal("a format no derivation produces should fail validation")
	}
}
