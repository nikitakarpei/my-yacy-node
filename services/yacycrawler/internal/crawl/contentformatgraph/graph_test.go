package contentformatgraph

import (
	"testing"
)

func TestDerivableAcceptsReachableFormat(t *testing.T) {
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
	if !graph.Derivable(FormatDocumentHTML, FormatMarkdown) {
		t.Fatal("markdown is reachable from document-html")
	}
}

func TestDerivableRejectsUnproducedFormat(t *testing.T) {
	graph := New(nil)
	if graph.Derivable(FormatDocumentHTML, FormatReadableText) {
		t.Fatal("a format no derivation produces is not derivable")
	}
}
