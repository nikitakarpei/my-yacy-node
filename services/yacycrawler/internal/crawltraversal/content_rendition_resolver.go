package crawltraversal

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type contentRenditionResolver struct {
	pageURL      string
	derivations  map[crawlcapability.PageContentFormat][]crawlcapability.PageRendering
	contents     map[crawlcapability.PageContentFormat][]byte
	unresolvable map[crawlcapability.PageContentFormat]bool
	resolving    map[crawlcapability.PageContentFormat]bool
}

func newContentRenditionResolver(
	page crawlcapability.CrawledPage,
	renderings []crawlcapability.PageRendering,
) *contentRenditionResolver {
	derivations := make(
		map[crawlcapability.PageContentFormat][]crawlcapability.PageRendering,
		len(renderings),
	)
	for _, rendering := range renderings {
		derivations[rendering.Format()] = append(derivations[rendering.Format()], rendering)
	}
	return &contentRenditionResolver{
		pageURL:      page.CanonicalURL,
		derivations:  derivations,
		contents:     map[crawlcapability.PageContentFormat][]byte{page.Format: page.Body},
		unresolvable: make(map[crawlcapability.PageContentFormat]bool),
		resolving:    make(map[crawlcapability.PageContentFormat]bool),
	}
}

func (r *contentRenditionResolver) resolve(
	format crawlcapability.PageContentFormat,
) ([]byte, bool, error) {
	if content, done := r.contents[format]; done {
		return content, true, nil
	}
	if r.unresolvable[format] || r.resolving[format] {
		return nil, false, nil
	}
	r.resolving[format] = true
	defer delete(r.resolving, format)
	for _, derivation := range r.derivations[format] {
		source, ready, err := r.resolve(derivation.SourceFormat())
		if err != nil {
			return nil, false, err
		}
		if !ready {
			continue
		}
		content, err := derivation.Render(r.pageURL, source)
		if err != nil {
			if errors.Is(err, crawlcapability.ErrUnextractable) {
				continue
			}
			return nil, false, fmt.Errorf("render %s: %w", format, err)
		}
		r.contents[format] = content
		return content, true, nil
	}
	r.unresolvable[format] = true
	return nil, false, nil
}
