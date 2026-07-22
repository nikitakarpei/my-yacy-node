package contentformatgraph

import (
	"fmt"
	"testing"
)

type scriptedDerivation struct {
	source Format
	target Format
	derive func(body []byte) ([]byte, error)
}

func (d scriptedDerivation) SourceFormat() Format { return d.source }
func (d scriptedDerivation) TargetFormat() Format { return d.target }

func (d scriptedDerivation) Derive(_ string, body []byte) ([]byte, error) {
	return d.derive(body)
}

func passthrough(tag string) func([]byte) ([]byte, error) {
	return func(body []byte) ([]byte, error) {
		return append([]byte(tag+":"), body...), nil
	}
}

func unextractable(_ []byte) ([]byte, error) {
	return nil, fmt.Errorf("%w: empty", ErrUnextractable)
}

func resolverFor(derivations ...Derivation) *Resolver {
	return New(derivations).Resolver(
		"http://host/",
		FormatDocumentHTML,
		[]byte("document"),
	)
}

func TestResolvePrefersEarlierCandidateOverFallback(t *testing.T) {
	resolver := resolverFor(
		scriptedDerivation{
			source: FormatDocumentHTML,
			target: FormatFullText,
			derive: passthrough("full"),
		},
		scriptedDerivation{
			source: FormatDocumentHTML,
			target: FormatReadableHTML,
			derive: passthrough("readable-html"),
		},
		scriptedDerivation{
			source: FormatReadableHTML,
			target: FormatReadableText,
			derive: passthrough("readable-text"),
		},
		scriptedDerivation{
			source: FormatFullText,
			target: FormatReadableText,
			derive: passthrough("fallback"),
		},
	)

	content, ready, err := resolver.Resolve(FormatReadableText)
	if err != nil || !ready {
		t.Fatalf("resolve readable-text: ready=%v err=%v", ready, err)
	}
	if got := string(content); got != "readable-text:readable-html:document" {
		t.Fatalf("readable-text derived via fallback, not readable-html: %q", got)
	}
	cached := resolver.Contents()[FormatReadableText]
	if string(cached) != string(content) {
		t.Fatalf("Contents omits the resolved readable-text: %q", cached)
	}
}

func TestResolveFallsBackWhenPreferredCandidateUnextractable(t *testing.T) {
	resolver := resolverFor(
		scriptedDerivation{
			source: FormatDocumentHTML,
			target: FormatFullText,
			derive: passthrough("full"),
		},
		scriptedDerivation{
			source: FormatDocumentHTML,
			target: FormatReadableHTML,
			derive: unextractable,
		},
		scriptedDerivation{
			source: FormatReadableHTML,
			target: FormatReadableText,
			derive: passthrough("readable-text"),
		},
		scriptedDerivation{
			source: FormatFullText,
			target: FormatReadableText,
			derive: passthrough("fallback"),
		},
	)

	content, ready, err := resolver.Resolve(FormatReadableText)
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
			source: FormatDocumentHTML,
			target: FormatReadableHTML,
			derive: unextractable,
		},
		scriptedDerivation{
			source: FormatReadableHTML,
			target: FormatReadableText,
			derive: passthrough("readable-text"),
		},
	)

	_, ready, err := resolver.Resolve(FormatReadableText)
	if err != nil {
		t.Fatalf("resolve readable-text: %v", err)
	}
	if ready {
		t.Fatal("readable-text should stay unresolved when every candidate is inapplicable")
	}
}
