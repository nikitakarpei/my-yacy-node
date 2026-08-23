package pageformats

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

type DerivableFormats struct {
	byTargetFormat map[documentextraction.Format][]FormatDerivation
}

func DerivableFormatsOf(derivations []FormatDerivation) DerivableFormats {
	byTargetFormat := make(
		map[documentextraction.Format][]FormatDerivation,
		len(derivations),
	)
	for _, derivation := range derivations {
		byTargetFormat[derivation.TargetFormat()] = append(
			byTargetFormat[derivation.TargetFormat()],
			derivation,
		)
	}
	return DerivableFormats{byTargetFormat: byTargetFormat}
}

func (f DerivableFormats) BodyIn(
	format documentextraction.Format,
	document documentextraction.Document,
	pageURL canonicalurl.CanonicalURL,
) ([]byte, bool, error) {
	return f.bodyIn(
		format,
		pageURL,
		map[documentextraction.Format][]byte{document.Format: document.Body},
		map[documentextraction.Format]bool{},
	)
}

func (f DerivableFormats) bodyIn(
	format documentextraction.Format,
	pageURL canonicalurl.CanonicalURL,
	derivedBodies map[documentextraction.Format][]byte,
	pendingFormats map[documentextraction.Format]bool,
) ([]byte, bool, error) {
	if body, derived := derivedBodies[format]; derived {
		return body, true, nil
	}
	if pendingFormats[format] {
		return nil, false, nil
	}
	pendingFormats[format] = true
	defer delete(pendingFormats, format)
	for _, derivation := range f.byTargetFormat[format] {
		sourceBody, ready, err := f.bodyIn(
			derivation.SourceFormat(), pageURL, derivedBodies, pendingFormats,
		)
		if err != nil {
			return nil, false, err
		}
		if !ready {
			continue
		}
		body, derived, err := derivation.BodyFrom(pageURL, sourceBody)
		if err != nil {
			return nil, false, fmt.Errorf("derive %s: %w", format, err)
		}
		if !derived {
			continue
		}
		derivedBodies[format] = body
		return body, true, nil
	}
	return nil, false, nil
}

func (f DerivableFormats) TargetFormats() []documentextraction.Format {
	targetFormats := make([]documentextraction.Format, 0, len(f.byTargetFormat))
	for targetFormat := range f.byTargetFormat {
		targetFormats = append(targetFormats, targetFormat)
	}
	return targetFormats
}

func (f DerivableFormats) Derivable(sourceFormat, targetFormat documentextraction.Format) bool {
	return f.derivable(sourceFormat, targetFormat, map[documentextraction.Format]bool{})
}

func (f DerivableFormats) derivable(
	sourceFormat, format documentextraction.Format,
	pendingFormats map[documentextraction.Format]bool,
) bool {
	if format == sourceFormat {
		return true
	}
	if pendingFormats[format] {
		return false
	}
	pendingFormats[format] = true
	for _, derivation := range f.byTargetFormat[format] {
		if f.derivable(sourceFormat, derivation.SourceFormat(), pendingFormats) {
			return true
		}
	}
	return false
}

func (f DerivableFormats) EnsureNoDanglingFormat(
	sourceFormats, targetFormats []documentextraction.Format,
) error {
	for _, targetFormat := range targetFormats {
		if !f.derivableFromAny(sourceFormats, targetFormat) {
			return fmt.Errorf("no source format derives %s", targetFormat)
		}
	}
	for _, sourceFormat := range sourceFormats {
		if !f.derivesAny(sourceFormat, targetFormats) {
			return fmt.Errorf("%s derives no target format", sourceFormat)
		}
	}
	return nil
}

func (f DerivableFormats) derivableFromAny(
	sourceFormats []documentextraction.Format,
	targetFormat documentextraction.Format,
) bool {
	for _, sourceFormat := range sourceFormats {
		if f.Derivable(sourceFormat, targetFormat) {
			return true
		}
	}
	return false
}

func (f DerivableFormats) derivesAny(
	sourceFormat documentextraction.Format,
	targetFormats []documentextraction.Format,
) bool {
	for _, targetFormat := range targetFormats {
		if f.Derivable(sourceFormat, targetFormat) {
			return true
		}
	}
	return false
}
