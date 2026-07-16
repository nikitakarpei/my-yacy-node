package crawlcapability

import "fmt"

// RenderedContent renders a page body through each rendering at most once, so
// representations that share a rendering (e.g. rwi and text both wanting the page's
// text) do not each pay for their own parse.
type RenderedContent struct {
	body         []byte
	sourceFormat PageContentFormat
	cache        map[ContentRendering][]byte
}

type RenderContent func(ContentRendering) ([]byte, error)

func NewRenderedContent(body []byte, sourceFormat PageContentFormat) *RenderedContent {
	return &RenderedContent{
		body:         body,
		sourceFormat: sourceFormat,
		cache:        map[ContentRendering][]byte{},
	}
}

func (r *RenderedContent) In(rendering ContentRendering) ([]byte, error) {
	if cached, ok := r.cache[rendering]; ok {
		return cached, nil
	}
	rendered, err := rendering.Render(r.body, r.sourceFormat)
	if err != nil {
		return nil, fmt.Errorf("render %s: %w", rendering.Format(), err)
	}
	r.cache[rendering] = rendered
	return rendered, nil
}
