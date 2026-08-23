package pageformats_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
)

func TestDerivableAcceptsReachableFormat(t *testing.T) {
	graph := pageformats.FormatDerivationsOf([]pageformats.Derivation{
		scriptedDerivation{
			source: documentextraction.FormatDocumentHTML,
			target: documentextraction.FormatReadableHTML,
		},
		scriptedDerivation{
			source: documentextraction.FormatReadableHTML,
			target: documentextraction.FormatMarkdown,
		},
	})
	if !graph.Derivable(documentextraction.FormatDocumentHTML, documentextraction.FormatMarkdown) {
		t.Fatal("markdown is reachable from document-html")
	}
}

func TestDerivableRejectsUnproducedFormat(t *testing.T) {
	graph := pageformats.FormatDerivationsOf(nil)
	if graph.Derivable(
		documentextraction.FormatDocumentHTML,
		documentextraction.FormatReadableText,
	) {
		t.Fatal("a format no derivation produces is not derivable")
	}
}

func readableTextGraph() pageformats.FormatDerivations {
	return pageformats.FormatDerivationsOf([]pageformats.Derivation{
		scriptedDerivation{
			source: documentextraction.FormatDocumentHTML,
			target: documentextraction.FormatReadableText,
		},
	})
}

func TestEnsureNoDanglingFormatAcceptsLinkedFormats(t *testing.T) {
	if err := readableTextGraph().EnsureNoDanglingFormat(
		[]documentextraction.Format{documentextraction.FormatDocumentHTML},
		[]documentextraction.Format{documentextraction.FormatReadableText},
	); err != nil {
		t.Fatalf("linked source and target formats should pass, got %v", err)
	}
}

func TestEnsureNoDanglingFormatRejectsUnderivedTarget(t *testing.T) {
	if err := readableTextGraph().EnsureNoDanglingFormat(
		[]documentextraction.Format{documentextraction.FormatDocumentHTML},
		[]documentextraction.Format{
			documentextraction.FormatReadableText,
			documentextraction.FormatMarkdown,
		},
	); err == nil {
		t.Fatal("a target format no source format derives should fail")
	}
}

func TestEnsureNoDanglingFormatRejectsUnreadSource(t *testing.T) {
	if err := readableTextGraph().EnsureNoDanglingFormat(
		[]documentextraction.Format{
			documentextraction.FormatDocumentHTML,
			documentextraction.FormatFullText,
		},
		[]documentextraction.Format{documentextraction.FormatReadableText},
	); err == nil {
		t.Fatal("a source format that derives no target format should fail")
	}
}
