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
	derive func(body []byte) ([]byte, bool, error)
}

func (d scriptedDerivation) SourceFormat() documentextraction.Format { return d.source }
func (d scriptedDerivation) TargetFormat() documentextraction.Format { return d.target }

func (d scriptedDerivation) Derive(
	_ canonicalurl.CanonicalURL,
	body []byte,
) ([]byte, bool, error) {
	return d.derive(body)
}

func passthrough(tag string) func([]byte) ([]byte, bool, error) {
	return func(body []byte) ([]byte, bool, error) {
		return append([]byte(tag+":"), body...), true, nil
	}
}

func underivable(_ []byte) ([]byte, bool, error) {
	return nil, false, nil
}

func resolverFor(t *testing.T, derivations ...pageformats.Derivation) *pageformats.PageFormats {
	t.Helper()
	return pageformats.FormatDerivationsOf(derivations).ForPage(
		documentextraction.Document{
			Format: documentextraction.FormatDocumentHTML,
			Body:   []byte("document"),
		},
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	)
}

func TestResolvePrefersEarlierCandidateOverFallback(t *testing.T) {
	resolver := resolverFor(t,
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

	content, ready, err := resolver.Resolve(documentextraction.FormatReadableText)
	if err != nil || !ready {
		t.Fatalf("resolve readable-text: ready=%v err=%v", ready, err)
	}
	if got := string(content); got != "readable-text:readable-html:document" {
		t.Fatalf("readable-text derived via fallback, not readable-html: %q", got)
	}
	cached, ready, err := resolver.Resolve(documentextraction.FormatReadableText)
	if err != nil || !ready || string(cached) != string(content) {
		t.Fatalf("re-resolved readable-text differs: %q", cached)
	}
}

func TestResolveFallsBackWhenThePreferredCandidateDerivesNothing(t *testing.T) {
	resolver := resolverFor(t,
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

	content, ready, err := resolver.Resolve(documentextraction.FormatReadableText)
	if err != nil || !ready {
		t.Fatalf("resolve readable-text: ready=%v err=%v", ready, err)
	}
	if got := string(content); got != "fallback:full:document" {
		t.Fatalf("readable-text should fall back to full-text: %q", got)
	}
}

func TestResolveLeavesFormatUnresolvedWhenNoCandidateApplies(t *testing.T) {
	resolver := resolverFor(t,
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

	_, ready, err := resolver.Resolve(documentextraction.FormatReadableText)
	if err != nil {
		t.Fatalf("resolve readable-text: %v", err)
	}
	if ready {
		t.Fatal("readable-text should stay unresolved when every candidate is inapplicable")
	}
}
