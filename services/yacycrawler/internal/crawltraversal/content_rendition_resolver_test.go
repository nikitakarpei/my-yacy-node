package crawltraversal

import (
	"fmt"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type scriptedRendering struct {
	source crawlcapability.PageContentFormat
	format crawlcapability.PageContentFormat
	render func(body []byte) ([]byte, error)
}

func (r scriptedRendering) SourceFormat() crawlcapability.PageContentFormat { return r.source }
func (r scriptedRendering) Format() crawlcapability.PageContentFormat       { return r.format }

func (r scriptedRendering) Render(
	_ string,
	body []byte,
) ([]byte, error) {
	return r.render(body)
}

func passthrough(tag string) func([]byte) ([]byte, error) {
	return func(body []byte) ([]byte, error) {
		return append([]byte(tag+":"), body...), nil
	}
}

func unextractable(_ []byte) ([]byte, error) {
	return nil, fmt.Errorf("%w: empty", crawlcapability.ErrUnextractable)
}

func htmlPage() crawlcapability.CrawledPage {
	return crawlcapability.CrawledPage{
		CanonicalURL: "http://host/",
		Body:         []byte("document"),
		Format:       crawlcapability.PageContentFormatDocumentHTML,
	}
}

func TestResolvePrefersEarlierCandidateOverFallback(t *testing.T) {
	resolver := newContentRenditionResolver(htmlPage(), []crawlcapability.PageRendering{
		scriptedRendering{
			source: crawlcapability.PageContentFormatDocumentHTML,
			format: crawlcapability.PageContentFormatFullText,
			render: passthrough("full"),
		},
		scriptedRendering{
			source: crawlcapability.PageContentFormatDocumentHTML,
			format: crawlcapability.PageContentFormatReadableHTML,
			render: passthrough("readable-html"),
		},
		scriptedRendering{
			source: crawlcapability.PageContentFormatReadableHTML,
			format: crawlcapability.PageContentFormatReadableText,
			render: passthrough("readable-text"),
		},
		scriptedRendering{
			source: crawlcapability.PageContentFormatFullText,
			format: crawlcapability.PageContentFormatReadableText,
			render: passthrough("fallback"),
		},
	})

	content, ready, err := resolver.resolve(crawlcapability.PageContentFormatReadableText)
	if err != nil || !ready {
		t.Fatalf("resolve readable-text: ready=%v err=%v", ready, err)
	}
	if got := string(content); got != "readable-text:readable-html:document" {
		t.Fatalf("readable-text derived via fallback, not readable-html: %q", got)
	}
}

func TestResolveFallsBackWhenPreferredCandidateUnextractable(t *testing.T) {
	resolver := newContentRenditionResolver(htmlPage(), []crawlcapability.PageRendering{
		scriptedRendering{
			source: crawlcapability.PageContentFormatDocumentHTML,
			format: crawlcapability.PageContentFormatFullText,
			render: passthrough("full"),
		},
		scriptedRendering{
			source: crawlcapability.PageContentFormatDocumentHTML,
			format: crawlcapability.PageContentFormatReadableHTML,
			render: unextractable,
		},
		scriptedRendering{
			source: crawlcapability.PageContentFormatReadableHTML,
			format: crawlcapability.PageContentFormatReadableText,
			render: passthrough("readable-text"),
		},
		scriptedRendering{
			source: crawlcapability.PageContentFormatFullText,
			format: crawlcapability.PageContentFormatReadableText,
			render: passthrough("fallback"),
		},
	})

	content, ready, err := resolver.resolve(crawlcapability.PageContentFormatReadableText)
	if err != nil || !ready {
		t.Fatalf("resolve readable-text: ready=%v err=%v", ready, err)
	}
	if got := string(content); got != "fallback:full:document" {
		t.Fatalf("readable-text should fall back to full-text: %q", got)
	}
}

func TestResolveLeavesRenditionUnresolvedWhenNoCandidateApplies(t *testing.T) {
	resolver := newContentRenditionResolver(htmlPage(), []crawlcapability.PageRendering{
		scriptedRendering{
			source: crawlcapability.PageContentFormatDocumentHTML,
			format: crawlcapability.PageContentFormatReadableHTML,
			render: unextractable,
		},
		scriptedRendering{
			source: crawlcapability.PageContentFormatReadableHTML,
			format: crawlcapability.PageContentFormatReadableText,
			render: passthrough("readable-text"),
		},
	})

	_, ready, err := resolver.resolve(crawlcapability.PageContentFormatReadableText)
	if err != nil {
		t.Fatalf("resolve readable-text: %v", err)
	}
	if ready {
		t.Fatal("readable-text should stay unresolved when every candidate is inapplicable")
	}
}
