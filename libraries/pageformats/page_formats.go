package pageformats

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

type PageFormats struct {
	pageURL      canonicalurl.CanonicalURL
	graph        FormatDerivations
	contents     map[documentextraction.Format][]byte
	unresolvable map[documentextraction.Format]bool
	resolving    map[documentextraction.Format]bool
}

func (r *PageFormats) Resolve(
	format documentextraction.Format,
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
		content, derived, err := derivation.Derive(r.pageURL, source)
		if err != nil {
			return nil, false, fmt.Errorf("derive %s: %w", format, err)
		}
		if !derived {
			continue
		}
		r.contents[format] = content
		return content, true, nil
	}
	r.unresolvable[format] = true
	return nil, false, nil
}
