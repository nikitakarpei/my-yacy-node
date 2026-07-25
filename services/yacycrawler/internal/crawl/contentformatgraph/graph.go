// Package contentformatgraph derives one content format from another along registered derivations.
package contentformatgraph

import (
	"fmt"
)

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

func (g FormatDerivations) EnsureDerivable(sourceFormat Format, targetFormats []Format) error {
	for _, format := range targetFormats {
		if !g.derivable(sourceFormat, format, map[Format]bool{}) {
			return fmt.Errorf("%s content is read but no derivation produces it", format)
		}
	}
	return nil
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
