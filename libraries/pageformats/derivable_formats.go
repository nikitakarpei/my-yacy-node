package pageformats

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

type DerivableFormats struct {
	byTargetFormat map[documentextraction.Format][]formatDerivation
}

func derivableFormatsOf(derivations []formatDerivation) DerivableFormats {
	byTargetFormat := make(
		map[documentextraction.Format][]formatDerivation,
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
	if format == document.Format {
		return document.Body, true, nil
	}
	for _, derivation := range f.byTargetFormat[format] {
		sourceBody, sourceDerived, err := f.BodyIn(
			derivation.SourceFormat(), document, pageURL,
		)
		if err != nil {
			return nil, false, err
		}
		if !sourceDerived {
			continue
		}
		body, derived, err := derivation.BodyFrom(pageURL, sourceBody)
		if err != nil {
			return nil, false, fmt.Errorf("derive %s: %w", format, err)
		}
		if !derived {
			continue
		}
		return body, true, nil
	}
	return nil, false, nil
}

func (f DerivableFormats) targetFormats() []documentextraction.Format {
	targetFormats := make([]documentextraction.Format, 0, len(f.byTargetFormat))
	for targetFormat := range f.byTargetFormat {
		targetFormats = append(targetFormats, targetFormat)
	}
	return targetFormats
}

func (f DerivableFormats) ensureNoDerivationCycle() error {
	for targetFormat := range f.byTargetFormat {
		if err := f.ensureNoDerivationCycleFrom(
			targetFormat, map[documentextraction.Format]bool{},
		); err != nil {
			return err
		}
	}
	return nil
}

func (f DerivableFormats) ensureNoDerivationCycleFrom(
	format documentextraction.Format,
	pendingFormats map[documentextraction.Format]bool,
) error {
	if pendingFormats[format] {
		return fmt.Errorf("%s derives from itself", format)
	}
	pendingFormats[format] = true
	defer delete(pendingFormats, format)
	for _, derivation := range f.byTargetFormat[format] {
		if err := f.ensureNoDerivationCycleFrom(
			derivation.SourceFormat(), pendingFormats,
		); err != nil {
			return err
		}
	}
	return nil
}

func (f DerivableFormats) ensureNoDanglingFormat(
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
		if f.derivable(sourceFormat, targetFormat) {
			return true
		}
	}
	return false
}

func (f DerivableFormats) derivable(
	sourceFormat, format documentextraction.Format,
) bool {
	if format == sourceFormat {
		return true
	}
	for _, derivation := range f.byTargetFormat[format] {
		if f.derivable(sourceFormat, derivation.SourceFormat()) {
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
		if f.derivable(sourceFormat, targetFormat) {
			return true
		}
	}
	return false
}
