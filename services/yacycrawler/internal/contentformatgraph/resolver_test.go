package contentformatgraph

import (
	"fmt"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type scriptedDerivation struct {
	source crawlcapability.PageContentFormat
	target crawlcapability.PageContentFormat
	derive func(body []byte) ([]byte, error)
}

func (d scriptedDerivation) SourceFormat() crawlcapability.PageContentFormat { return d.source }
func (d scriptedDerivation) TargetFormat() crawlcapability.PageContentFormat { return d.target }

func (d scriptedDerivation) Derive(_ string, body []byte) ([]byte, error) {
	return d.derive(body)
}

func passthrough(tag string) func([]byte) ([]byte, error) {
	return func(body []byte) ([]byte, error) {
		return append([]byte(tag+":"), body...), nil
	}
}

func unextractable(_ []byte) ([]byte, error) {
	return nil, fmt.Errorf("%w: empty", crawlcapability.ErrUnextractable)
}

func resolverFor(derivations ...crawlcapability.PageDerivation) *Resolver {
	return New(derivations).Resolver(
		"http://host/",
		crawlcapability.PageContentFormatDocumentHTML,
		[]byte("document"),
	)
}

func TestResolvePrefersEarlierCandidateOverFallback(t *testing.T) {
	resolver := resolverFor(
		scriptedDerivation{
			source: crawlcapability.PageContentFormatDocumentHTML,
			target: crawlcapability.PageContentFormatFullText,
			derive: passthrough("full"),
		},
		scriptedDerivation{
			source: crawlcapability.PageContentFormatDocumentHTML,
			target: crawlcapability.PageContentFormatReadableHTML,
			derive: passthrough("readable-html"),
		},
		scriptedDerivation{
			source: crawlcapability.PageContentFormatReadableHTML,
			target: crawlcapability.PageContentFormatReadableText,
			derive: passthrough("readable-text"),
		},
		scriptedDerivation{
			source: crawlcapability.PageContentFormatFullText,
			target: crawlcapability.PageContentFormatReadableText,
			derive: passthrough("fallback"),
		},
	)

	content, ready, err := resolver.Resolve(crawlcapability.PageContentFormatReadableText)
	if err != nil || !ready {
		t.Fatalf("resolve readable-text: ready=%v err=%v", ready, err)
	}
	if got := string(content); got != "readable-text:readable-html:document" {
		t.Fatalf("readable-text derived via fallback, not readable-html: %q", got)
	}
	cached := resolver.Contents()[crawlcapability.PageContentFormatReadableText]
	if string(cached) != string(content) {
		t.Fatalf("Contents omits the resolved readable-text: %q", cached)
	}
}

func TestResolveFallsBackWhenPreferredCandidateUnextractable(t *testing.T) {
	resolver := resolverFor(
		scriptedDerivation{
			source: crawlcapability.PageContentFormatDocumentHTML,
			target: crawlcapability.PageContentFormatFullText,
			derive: passthrough("full"),
		},
		scriptedDerivation{
			source: crawlcapability.PageContentFormatDocumentHTML,
			target: crawlcapability.PageContentFormatReadableHTML,
			derive: unextractable,
		},
		scriptedDerivation{
			source: crawlcapability.PageContentFormatReadableHTML,
			target: crawlcapability.PageContentFormatReadableText,
			derive: passthrough("readable-text"),
		},
		scriptedDerivation{
			source: crawlcapability.PageContentFormatFullText,
			target: crawlcapability.PageContentFormatReadableText,
			derive: passthrough("fallback"),
		},
	)

	content, ready, err := resolver.Resolve(crawlcapability.PageContentFormatReadableText)
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
			source: crawlcapability.PageContentFormatDocumentHTML,
			target: crawlcapability.PageContentFormatReadableHTML,
			derive: unextractable,
		},
		scriptedDerivation{
			source: crawlcapability.PageContentFormatReadableHTML,
			target: crawlcapability.PageContentFormatReadableText,
			derive: passthrough("readable-text"),
		},
	)

	_, ready, err := resolver.Resolve(crawlcapability.PageContentFormatReadableText)
	if err != nil {
		t.Fatalf("resolve readable-text: %v", err)
	}
	if ready {
		t.Fatal("readable-text should stay unresolved when every candidate is inapplicable")
	}
}
