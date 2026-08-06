// Package contentformatgraph derives one content format from another along registered derivations.
package contentformatgraph

type FormatDerivations struct {
	byTargetFormat map[Format][]Derivation
}

func New(derivations []Derivation) FormatDerivations {
	byTargetFormat := make(
		map[Format][]Derivation,
		len(derivations),
	)
	for _, derivation := range derivations {
		byTargetFormat[derivation.TargetFormat()] = append(
			byTargetFormat[derivation.TargetFormat()],
			derivation,
		)
	}
	return FormatDerivations{byTargetFormat: byTargetFormat}
}

func (g FormatDerivations) Derivable(sourceFormat, targetFormat Format) bool {
	return g.derivable(sourceFormat, targetFormat, map[Format]bool{})
}

func (g FormatDerivations) derivable(sourceFormat, format Format, resolving map[Format]bool) bool {
	if format == sourceFormat {
		return true
	}
	if resolving[format] {
		return false
	}
	resolving[format] = true
	for _, derivation := range g.byTargetFormat[format] {
		if g.derivable(sourceFormat, derivation.SourceFormat(), resolving) {
			return true
		}
	}
	return false
}

func (g FormatDerivations) ForPage(
	pageURL string,
	format Format,
	body []byte,
) *PageFormats {
	return &PageFormats{
		pageURL:      pageURL,
		graph:        g,
		contents:     map[Format][]byte{format: body},
		unresolvable: make(map[Format]bool),
		resolving:    make(map[Format]bool),
	}
}
