package crawlcapability

import "fmt"

// RenderedContent renders a page body into each requested target format at most once,
// so representations that share a target (e.g. rwi and text both wanting the page's
// text) do not each pay for their own parse.
type RenderedContent struct {
	body         []byte
	sourceFormat PageContentFormat
	cache        map[PageContentFormat][]byte
}

func NewRenderedContent(body []byte, sourceFormat PageContentFormat) *RenderedContent {
	return &RenderedContent{
		body:         body,
		sourceFormat: sourceFormat,
		cache:        map[PageContentFormat][]byte{},
	}
}

func (r *RenderedContent) In(rendering ContentRendering) ([]byte, error) {
	format := rendering.Format()
	if cached, ok := r.cache[format]; ok {
		return cached, nil
	}
	rendered, err := rendering.Render(r.body, r.sourceFormat)
	if err != nil {
		return nil, fmt.Errorf("render %s: %w", format, err)
	}
	r.cache[format] = rendered
	return rendered, nil
}
