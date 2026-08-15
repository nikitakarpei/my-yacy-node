package contentformatgraph_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
)

func TestDerivableAcceptsReachableFormat(t *testing.T) {
	graph := contentformatgraph.New([]contentformatgraph.Derivation{
		scriptedDerivation{
			source: contentformatgraph.FormatDocumentHTML,
			target: contentformatgraph.FormatReadableHTML,
		},
		scriptedDerivation{
			source: contentformatgraph.FormatReadableHTML,
			target: contentformatgraph.FormatMarkdown,
		},
	})
	if !graph.Derivable(contentformatgraph.FormatDocumentHTML, contentformatgraph.FormatMarkdown) {
		t.Fatal("markdown is reachable from document-html")
	}
}

func TestDerivableRejectsUnproducedFormat(t *testing.T) {
	graph := contentformatgraph.New(nil)
	if graph.Derivable(
		contentformatgraph.FormatDocumentHTML,
		contentformatgraph.FormatReadableText,
	) {
		t.Fatal("a format no derivation produces is not derivable")
	}
}

func readableTextGraph() contentformatgraph.FormatDerivations {
	return contentformatgraph.New([]contentformatgraph.Derivation{
		scriptedDerivation{
			source: contentformatgraph.FormatDocumentHTML,
			target: contentformatgraph.FormatReadableText,
		},
	})
}

func TestEnsureNoDanglingFormatAcceptsLinkedFormats(t *testing.T) {
	if err := readableTextGraph().EnsureNoDanglingFormat(
		[]contentformatgraph.Format{contentformatgraph.FormatDocumentHTML},
		[]contentformatgraph.Format{contentformatgraph.FormatReadableText},
	); err != nil {
		t.Fatalf("linked source and target formats should pass, got %v", err)
	}
}

func TestEnsureNoDanglingFormatRejectsUnderivedTarget(t *testing.T) {
	if err := readableTextGraph().EnsureNoDanglingFormat(
		[]contentformatgraph.Format{contentformatgraph.FormatDocumentHTML},
		[]contentformatgraph.Format{
			contentformatgraph.FormatReadableText,
			contentformatgraph.FormatMarkdown,
		},
	); err == nil {
		t.Fatal("a target format no source format derives should fail")
	}
}

func TestEnsureNoDanglingFormatRejectsUnreadSource(t *testing.T) {
	if err := readableTextGraph().EnsureNoDanglingFormat(
		[]contentformatgraph.Format{
			contentformatgraph.FormatDocumentHTML,
			contentformatgraph.FormatFullText,
		},
		[]contentformatgraph.Format{contentformatgraph.FormatReadableText},
	); err == nil {
		t.Fatal("a source format that derives no target format should fail")
	}
}
