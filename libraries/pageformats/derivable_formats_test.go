package pageformats_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
)

type scriptedDerivation struct {
	source documentextraction.Format
	target documentextraction.Format
	derive func(sourceBody []byte) ([]byte, bool, error)
}

func (d scriptedDerivation) SourceFormat() documentextraction.Format { return d.source }
func (d scriptedDerivation) TargetFormat() documentextraction.Format { return d.target }

func (d scriptedDerivation) BodyFrom(
	_ canonicalurl.CanonicalURL,
	sourceBody []byte,
) ([]byte, bool, error) {
	return d.derive(sourceBody)
}

func passthrough(tag string) func([]byte) ([]byte, bool, error) {
	return func(sourceBody []byte) ([]byte, bool, error) {
		return append([]byte(tag+":"), sourceBody...), true, nil
	}
}

func underivable(_ []byte) ([]byte, bool, error) {
	return nil, false, nil
}

func bodyIn(
	t *testing.T,
	format documentextraction.Format,
	derivations ...pageformats.FormatDerivation,
) ([]byte, bool, error) {
	t.Helper()
	return pageformats.DerivableFormatsOf(derivations).BodyIn(
		format,
		documentextraction.Document{
			Format: documentextraction.FormatDocumentHTML,
			Body:   []byte("document"),
		},
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	)
}

func TestBodyInPrefersEarlierCandidateOverFallback(t *testing.T) {
	body, derived, err := bodyIn(t, documentextraction.FormatReadableText,
		scriptedDerivation{
			source: documentextraction.FormatDocumentHTML,
			target: documentextraction.FormatFullText,
			derive: passthrough("full"),
		},
		scriptedDerivation{
			source: documentextraction.FormatDocumentHTML,
			target: documentextraction.FormatReadableHTML,
			derive: passthrough("readable-html"),
		},
		scriptedDerivation{
			source: documentextraction.FormatReadableHTML,
			target: documentextraction.FormatReadableText,
			derive: passthrough("readable-text"),
		},
		scriptedDerivation{
			source: documentextraction.FormatFullText,
			target: documentextraction.FormatReadableText,
			derive: passthrough("fallback"),
		},
	)
	if err != nil || !derived {
		t.Fatalf("readable-text: derived=%v err=%v", derived, err)
	}
	if got := string(body); got != "readable-text:readable-html:document" {
		t.Fatalf("readable-text derived via fallback, not readable-html: %q", got)
	}
}

func TestBodyInFallsBackWhenThePreferredCandidateDerivesNothing(t *testing.T) {
	body, derived, err := bodyIn(t, documentextraction.FormatReadableText,
		scriptedDerivation{
			source: documentextraction.FormatDocumentHTML,
			target: documentextraction.FormatFullText,
			derive: passthrough("full"),
		},
		scriptedDerivation{
			source: documentextraction.FormatDocumentHTML,
			target: documentextraction.FormatReadableHTML,
			derive: underivable,
		},
		scriptedDerivation{
			source: documentextraction.FormatReadableHTML,
			target: documentextraction.FormatReadableText,
			derive: passthrough("readable-text"),
		},
		scriptedDerivation{
			source: documentextraction.FormatFullText,
			target: documentextraction.FormatReadableText,
			derive: passthrough("fallback"),
		},
	)
	if err != nil || !derived {
		t.Fatalf("readable-text: derived=%v err=%v", derived, err)
	}
	if got := string(body); got != "fallback:full:document" {
		t.Fatalf("readable-text should fall back to full-text: %q", got)
	}
}

func TestBodyInDerivesNothingWhenNoCandidateApplies(t *testing.T) {
	_, derived, err := bodyIn(t, documentextraction.FormatReadableText,
		scriptedDerivation{
			source: documentextraction.FormatDocumentHTML,
			target: documentextraction.FormatReadableHTML,
			derive: underivable,
		},
		scriptedDerivation{
			source: documentextraction.FormatReadableHTML,
			target: documentextraction.FormatReadableText,
			derive: passthrough("readable-text"),
		},
	)
	if err != nil {
		t.Fatalf("readable-text: %v", err)
	}
	if derived {
		t.Fatal("readable-text should derive nothing when every candidate is inapplicable")
	}
}

func TestBodyInReturnsTheDocumentBodyInItsOwnFormat(t *testing.T) {
	body, derived, err := bodyIn(t, documentextraction.FormatDocumentHTML)
	if err != nil || !derived {
		t.Fatalf("document-html: derived=%v err=%v", derived, err)
	}
	if got := string(body); got != "document" {
		t.Fatalf("document-html body: %q", got)
	}
}

func TestBodyInDerivesNothingThroughACycle(t *testing.T) {
	_, derived, err := bodyIn(t, documentextraction.FormatMarkdown,
		scriptedDerivation{
			source: documentextraction.FormatReadableText,
			target: documentextraction.FormatMarkdown,
			derive: passthrough("markdown"),
		},
		scriptedDerivation{
			source: documentextraction.FormatMarkdown,
			target: documentextraction.FormatReadableText,
			derive: passthrough("readable-text"),
		},
	)
	if err != nil {
		t.Fatalf("markdown: %v", err)
	}
	if derived {
		t.Fatal("a cycle that reaches no document format derives nothing")
	}
}

func TestDerivableAcceptsReachableFormat(t *testing.T) {
	derivableFormats := pageformats.DerivableFormatsOf([]pageformats.FormatDerivation{
		scriptedDerivation{
			source: documentextraction.FormatDocumentHTML,
			target: documentextraction.FormatReadableHTML,
		},
		scriptedDerivation{
			source: documentextraction.FormatReadableHTML,
			target: documentextraction.FormatMarkdown,
		},
	})
	if !derivableFormats.Derivable(
		documentextraction.FormatDocumentHTML,
		documentextraction.FormatMarkdown,
	) {
		t.Fatal("markdown is reachable from document-html")
	}
}

func TestDerivableRejectsUnproducedFormat(t *testing.T) {
	derivableFormats := pageformats.DerivableFormatsOf(nil)
	if derivableFormats.Derivable(
		documentextraction.FormatDocumentHTML,
		documentextraction.FormatReadableText,
	) {
		t.Fatal("a format no derivation produces is not derivable")
	}
}

func readableTextFormats() pageformats.DerivableFormats {
	return pageformats.DerivableFormatsOf([]pageformats.FormatDerivation{
		scriptedDerivation{
			source: documentextraction.FormatDocumentHTML,
			target: documentextraction.FormatReadableText,
		},
	})
}

func TestEnsureNoDanglingFormatAcceptsLinkedFormats(t *testing.T) {
	if err := readableTextFormats().EnsureNoDanglingFormat(
		[]documentextraction.Format{documentextraction.FormatDocumentHTML},
		[]documentextraction.Format{documentextraction.FormatReadableText},
	); err != nil {
		t.Fatalf("linked source and target formats should pass, got %v", err)
	}
}

func TestEnsureNoDanglingFormatRejectsUnderivedTarget(t *testing.T) {
	if err := readableTextFormats().EnsureNoDanglingFormat(
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
	if err := readableTextFormats().EnsureNoDanglingFormat(
		[]documentextraction.Format{
			documentextraction.FormatDocumentHTML,
			documentextraction.FormatFullText,
		},
		[]documentextraction.Format{documentextraction.FormatReadableText},
	); err == nil {
		t.Fatal("a source format that derives no target format should fail")
	}
}
