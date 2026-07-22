package contentformatgraph

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type Resolver struct {
	pageURL      string
	graph        Graph
	contents     map[crawlcapability.PageContentFormat][]byte
	unresolvable map[crawlcapability.PageContentFormat]bool
	resolving    map[crawlcapability.PageContentFormat]bool
}

func (r *Resolver) Resolve(
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
	for _, derivation := range r.graph.byTargetFormat[format] {
		source, ready, err := r.Resolve(derivation.SourceFormat())
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

func (r *Resolver) Contents() map[crawlcapability.PageContentFormat][]byte {
	return r.contents
}
