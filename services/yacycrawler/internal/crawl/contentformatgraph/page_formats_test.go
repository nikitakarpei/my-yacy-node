package contentformatgraph_test

import (
	"fmt"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
)

type scriptedDerivation struct {
	source contentformatgraph.Format
	target contentformatgraph.Format
	derive func(body []byte) ([]byte, error)
}

func (d scriptedDerivation) SourceFormat() contentformatgraph.Format { return d.source }
func (d scriptedDerivation) TargetFormat() contentformatgraph.Format { return d.target }

func (d scriptedDerivation) Derive(_ string, body []byte) ([]byte, error) {
	return d.derive(body)
}

func passthrough(tag string) func([]byte) ([]byte, error) {
	return func(body []byte) ([]byte, error) {
		return append([]byte(tag+":"), body...), nil
	}
}

func unextractable(_ []byte) ([]byte, error) {
	return nil, fmt.Errorf("%w: empty", contentformatgraph.ErrUnderivable)
}

func resolverFor(derivations ...contentformatgraph.Derivation) *contentformatgraph.PageFormats {
	return contentformatgraph.New(derivations).ForPage(
		"http://host/",
		contentformatgraph.FormatDocumentHTML,
		[]byte("document"),
	)
}

func TestResolvePrefersEarlierCandidateOverFallback(t *testing.T) {
	resolver := resolverFor(
		scriptedDerivation{
			source: contentformatgraph.FormatDocumentHTML,
			target: contentformatgraph.FormatFullText,
			derive: passthrough("full"),
		},
		scriptedDerivation{
			source: contentformatgraph.FormatDocumentHTML,
			target: contentformatgraph.FormatReadableHTML,
			derive: passthrough("readable-html"),
		},
		scriptedDerivation{
			source: contentformatgraph.FormatReadableHTML,
			target: contentformatgraph.FormatReadableText,
			derive: passthrough("readable-text"),
		},
		scriptedDerivation{
			source: contentformatgraph.FormatFullText,
			target: contentformatgraph.FormatReadableText,
			derive: passthrough("fallback"),
		},
	)

	content, ready, err := resolver.Resolve(contentformatgraph.FormatReadableText)
	if err != nil || !ready {
		t.Fatalf("resolve readable-text: ready=%v err=%v", ready, err)
	}
	if got := string(content); got != "readable-text:readable-html:document" {
		t.Fatalf("readable-text derived via fallback, not readable-html: %q", got)
	}
	cached, ready, err := resolver.Resolve(contentformatgraph.FormatReadableText)
	if err != nil || !ready || string(cached) != string(content) {
		t.Fatalf("re-resolved readable-text differs: %q", cached)
	}
}

func TestResolveFallsBackWhenPreferredCandidateUnextractable(t *testing.T) {
	resolver := resolverFor(
		scriptedDerivation{
			source: contentformatgraph.FormatDocumentHTML,
			target: contentformatgraph.FormatFullText,
			derive: passthrough("full"),
		},
		scriptedDerivation{
			source: contentformatgraph.FormatDocumentHTML,
			target: contentformatgraph.FormatReadableHTML,
			derive: unextractable,
		},
		scriptedDerivation{
			source: contentformatgraph.FormatReadableHTML,
			target: contentformatgraph.FormatReadableText,
			derive: passthrough("readable-text"),
		},
		scriptedDerivation{
			source: contentformatgraph.FormatFullText,
			target: contentformatgraph.FormatReadableText,
			derive: passthrough("fallback"),
		},
	)

	content, ready, err := resolver.Resolve(contentformatgraph.FormatReadableText)
	if err != nil || !ready {
		t.Fatalf("resolve readable-text: ready=%v err=%v", ready, err)
	}
	if got := string(content); got != "fallback:full:document" {
		t.Fatalf("readable-text should fall back to full-text: %q", got)
	}
}

func TestResolveLeavesFormatUnresolvedWhenNoCandidateApplies(t *testing.T) {
	resolver := resolverFor(
		scriptedDerivation{
			source: contentformatgraph.FormatDocumentHTML,
			target: contentformatgraph.FormatReadableHTML,
			derive: unextractable,
		},
		scriptedDerivation{
			source: contentformatgraph.FormatReadableHTML,
			target: contentformatgraph.FormatReadableText,
			derive: passthrough("readable-text"),
		},
	)

	_, ready, err := resolver.Resolve(contentformatgraph.FormatReadableText)
	if err != nil {
		t.Fatalf("resolve readable-text: %v", err)
	}
	if ready {
		t.Fatal("readable-text should stay unresolved when every candidate is inapplicable")
	}
}
