package crawltraversal

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type contentFormatResolver struct {
	pageURL        string
	byTargetFormat map[crawlcapability.PageContentFormat][]crawlcapability.PageDerivation
	contents       map[crawlcapability.PageContentFormat][]byte
	unresolvable   map[crawlcapability.PageContentFormat]bool
	resolving      map[crawlcapability.PageContentFormat]bool
}

func newContentFormatResolver(
	page crawlcapability.CrawledPage,
	derivations []crawlcapability.PageDerivation,
) *contentFormatResolver {
	byTargetFormat := make(
		map[crawlcapability.PageContentFormat][]crawlcapability.PageDerivation,
		len(derivations),
	)
	for _, derivation := range derivations {
		byTargetFormat[derivation.TargetFormat()] = append(
			byTargetFormat[derivation.TargetFormat()],
			derivation,
		)
	}
	return &contentFormatResolver{
		pageURL:        page.CanonicalURL,
		byTargetFormat: byTargetFormat,
		contents:       map[crawlcapability.PageContentFormat][]byte{page.Format: page.Body},
		unresolvable:   make(map[crawlcapability.PageContentFormat]bool),
		resolving:      make(map[crawlcapability.PageContentFormat]bool),
	}
}

func (r *contentFormatResolver) resolve(
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
	for _, derivation := range r.byTargetFormat[format] {
		source, ready, err := r.resolve(derivation.SourceFormat())
		if err != nil {
			return nil, false, err
		}
		if !ready {
			continue
		}
		content, err := derivation.Derive(r.pageURL, source)
		if err != nil {
			if errors.Is(err, crawlcapability.ErrUnextractable) {
				continue
			}
			return nil, false, fmt.Errorf("derive %s: %w", format, err)
		}
		r.contents[format] = content
		return content, true, nil
	}
	r.unresolvable[format] = true
	return nil, false, nil
}
