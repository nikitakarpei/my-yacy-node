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

func readableTextGraph() FormatDerivations {
	return New([]Derivation{
		scriptedDerivation{source: FormatDocumentHTML, target: FormatReadableText},
	})
}

func TestEnsureNoDanglingFormatAcceptsLinkedFormats(t *testing.T) {
	if err := readableTextGraph().EnsureNoDanglingFormat(
		[]Format{FormatDocumentHTML},
		[]Format{FormatReadableText},
	); err != nil {
		t.Fatalf("linked source and target formats should pass, got %v", err)
	}
}

func TestEnsureNoDanglingFormatRejectsUnderivedTarget(t *testing.T) {
	if err := readableTextGraph().EnsureNoDanglingFormat(
		[]Format{FormatDocumentHTML},
		[]Format{FormatReadableText, FormatMarkdown},
	); err == nil {
		t.Fatal("a target format no source format derives should fail")
	}
}

func TestEnsureNoDanglingFormatRejectsUnreadSource(t *testing.T) {
	if err := readableTextGraph().EnsureNoDanglingFormat(
		[]Format{FormatDocumentHTML, FormatFullText},
		[]Format{FormatReadableText},
	); err == nil {
		t.Fatal("a source format that derives no target format should fail")
	}
}
