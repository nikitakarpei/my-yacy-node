package renderedcontent_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/renderedcontent"
)

type countingRendering struct {
	format crawlcapability.PageContentFormat
	calls  int
}

func (r *countingRendering) Format() crawlcapability.PageContentFormat { return r.format }

func (r *countingRendering) SourceFormats() []crawlcapability.PageContentFormat {
	return []crawlcapability.PageContentFormat{crawlcapability.PageContentFormatHTML}
}

func (r *countingRendering) Render(
	body []byte,
	_ crawlcapability.PageContentFormat,
) ([]byte, error) {
	r.calls++
	return body, nil
}

func TestRenderedContentRendersEachFormatOnce(t *testing.T) {
	rendering := &countingRendering{format: crawlcapability.PageContentFormatText}
	rendered := renderedcontent.New(
		[]byte("body"), crawlcapability.PageContentFormatHTML,
	)

	if _, err := rendered.In(rendering); err != nil {
		t.Fatalf("In: %v", err)
	}
	if _, err := rendered.In(rendering); err != nil {
		t.Fatalf("In: %v", err)
	}

	if rendering.calls != 1 {
		t.Fatalf("Render called %d times, want 1", rendering.calls)
	}
}

func TestRenderedContentRendersEachFormatIndependently(t *testing.T) {
	text := &countingRendering{format: crawlcapability.PageContentFormatText}
	markdown := &countingRendering{format: crawlcapability.PageContentFormatMarkdown}
	rendered := renderedcontent.New(
		[]byte("body"), crawlcapability.PageContentFormatHTML,
	)

	if _, err := rendered.In(text); err != nil {
		t.Fatalf("In: %v", err)
	}
	if _, err := rendered.In(markdown); err != nil {
		t.Fatalf("In: %v", err)
	}

	if text.calls != 1 || markdown.calls != 1 {
		t.Fatalf("expected one render each, got text=%d markdown=%d", text.calls, markdown.calls)
	}
}

type configuredRendering struct {
	stopWords []string
}

func (configuredRendering) Format() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatText
}

func (configuredRendering) SourceFormats() []crawlcapability.PageContentFormat {
	return []crawlcapability.PageContentFormat{crawlcapability.PageContentFormatHTML}
}

func (configuredRendering) Render(
	body []byte,
	_ crawlcapability.PageContentFormat,
) ([]byte, error) {
	return body, nil
}

func TestRenderedContentAcceptsARenderingCarryingConfiguration(t *testing.T) {
	rendered := renderedcontent.New(
		[]byte("body"), crawlcapability.PageContentFormatHTML,
	)

	if _, err := rendered.In(configuredRendering{stopWords: []string{"the"}}); err != nil {
		t.Fatalf("In: %v", err)
	}
}
