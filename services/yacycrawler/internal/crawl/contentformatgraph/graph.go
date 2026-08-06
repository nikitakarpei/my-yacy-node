// Package contentformatgraph derives one content format from another along registered derivations.
package contentformatgraph

import "fmt"

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

func (g FormatDerivations) EnsureNoDanglingFormat(sourceFormats, targetFormats []Format) error {
	for _, targetFormat := range targetFormats {
		if !g.derivableFromAny(sourceFormats, targetFormat) {
			return fmt.Errorf("no source format derives %s", targetFormat)
		}
	}
	for _, sourceFormat := range sourceFormats {
		if !g.derivesAny(sourceFormat, targetFormats) {
			return fmt.Errorf("%s derives no target format", sourceFormat)
		}
	}
	return nil
}

func (g FormatDerivations) derivableFromAny(sourceFormats []Format, targetFormat Format) bool {
	for _, sourceFormat := range sourceFormats {
		if g.Derivable(sourceFormat, targetFormat) {
			return true
		}
	}
	return false
}

func (g FormatDerivations) derivesAny(sourceFormat Format, targetFormats []Format) bool {
	for _, targetFormat := range targetFormats {
		if g.Derivable(sourceFormat, targetFormat) {
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
