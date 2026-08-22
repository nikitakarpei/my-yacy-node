// Package contentformatgraph derives one content format from another along registered derivations.
package contentformatgraph

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

type FormatDerivations struct {
	byTargetFormat map[documentextraction.Format][]Derivation
}

func New(derivations []Derivation) FormatDerivations {
	byTargetFormat := make(
		map[documentextraction.Format][]Derivation,
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

func (g FormatDerivations) TargetFormats() []documentextraction.Format {
	targetFormats := make([]documentextraction.Format, 0, len(g.byTargetFormat))
	for targetFormat := range g.byTargetFormat {
		targetFormats = append(targetFormats, targetFormat)
	}
	return targetFormats
}

func (g FormatDerivations) Derivable(sourceFormat, targetFormat documentextraction.Format) bool {
	return g.derivable(sourceFormat, targetFormat, map[documentextraction.Format]bool{})
}

func (g FormatDerivations) derivable(
	sourceFormat, format documentextraction.Format,
	resolving map[documentextraction.Format]bool,
) bool {
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

func (g FormatDerivations) EnsureNoDanglingFormat(
	sourceFormats, targetFormats []documentextraction.Format,
) error {
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

func (g FormatDerivations) derivableFromAny(
	sourceFormats []documentextraction.Format,
	targetFormat documentextraction.Format,
) bool {
	for _, sourceFormat := range sourceFormats {
		if g.Derivable(sourceFormat, targetFormat) {
			return true
		}
	}
	return false
}

func (g FormatDerivations) derivesAny(
	sourceFormat documentextraction.Format,
	targetFormats []documentextraction.Format,
) bool {
	for _, targetFormat := range targetFormats {
		if g.Derivable(sourceFormat, targetFormat) {
			return true
		}
	}
	return false
}

func (g FormatDerivations) ForPage(
	pageURL string,
	format documentextraction.Format,
	body []byte,
) *PageFormats {
	return &PageFormats{
		pageURL:      pageURL,
		graph:        g,
		contents:     map[documentextraction.Format][]byte{format: body},
		unresolvable: make(map[documentextraction.Format]bool),
		resolving:    make(map[documentextraction.Format]bool),
	}
}
