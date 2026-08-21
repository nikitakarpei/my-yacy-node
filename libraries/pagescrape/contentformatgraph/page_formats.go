package contentformatgraph

import (
	"errors"
	"fmt"
)

type PageFormats struct {
	pageURL      string
	graph        FormatDerivations
	contents     map[Format][]byte
	unresolvable map[Format]bool
	resolving    map[Format]bool
}

func (r *PageFormats) Resolve(
	format Format,
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
			if errors.Is(err, ErrUnderivable) {
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
