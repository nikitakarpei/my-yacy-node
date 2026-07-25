package contentformatgraph

import (
	"testing"
)

func TestValidateAcceptsReachableFormat(t *testing.T) {
	graph := New([]Derivation{
		scriptedDerivation{
			source: FormatDocumentHTML,
			target: FormatReadableHTML,
		},
		scriptedDerivation{
			source: FormatReadableHTML,
			target: FormatMarkdown,
		},
	})
	if err := graph.EnsureDerivable(FormatDocumentHTML, []Format{
		FormatMarkdown,
	}); err != nil {
		t.Fatalf("markdown is reachable from document-html: %v", err)
	}
}

func TestValidateRejectsUnproducedFormat(t *testing.T) {
	graph := New(nil)
	if err := graph.EnsureDerivable(FormatDocumentHTML, []Format{
		FormatReadableText,
	}); err == nil {
		t.Fatal("a format no derivation produces should fail validation")
	}
}
