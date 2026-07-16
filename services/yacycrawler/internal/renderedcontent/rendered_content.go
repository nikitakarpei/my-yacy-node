// Package renderedcontent renders a page body into each content format at most once, so
// representations that share a format (e.g. rwi and text both wanting the page's text)
// do not each pay for their own parse. A page has one body per format, so one rendering
// produces each.
package renderedcontent

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type Content struct {
	body         []byte
	sourceFormat crawlcapability.PageContentFormat
	cache        map[crawlcapability.PageContentFormat][]byte
}

func New(body []byte, sourceFormat crawlcapability.PageContentFormat) *Content {
	return &Content{
		body:         body,
		sourceFormat: sourceFormat,
		cache:        map[crawlcapability.PageContentFormat][]byte{},
	}
}

func (c *Content) In(rendering crawlcapability.ContentRendering) ([]byte, error) {
	format := rendering.Format()
	if cached, ok := c.cache[format]; ok {
		return cached, nil
	}
	rendered, err := rendering.Render(c.body, c.sourceFormat)
	if err != nil {
		return nil, fmt.Errorf("render %s: %w", format, err)
	}
	c.cache[format] = rendered
	return rendered, nil
}
