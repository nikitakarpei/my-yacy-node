// Package derivationreach reports which formats a set of format derivations can reach.
package derivationreach

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

type DerivationFormats struct {
	SourceFormat documentextraction.Format
	TargetFormat documentextraction.Format
}

func EnsureNoCycle(derivations []DerivationFormats) error {
	return derivationReachOf(derivations).ensureNoCycle()
}

func EnsureNoDanglingFormat(
	derivations []DerivationFormats,
	emittedFormats []documentextraction.Format,
) error {
	return derivationReachOf(derivations).ensureNoDanglingFormat(emittedFormats)
}

type derivationReach struct {
	sourceFormatsByTargetFormat map[documentextraction.Format][]documentextraction.Format
}

func derivationReachOf(derivations []DerivationFormats) derivationReach {
	sourceFormatsByTargetFormat := make(
		map[documentextraction.Format][]documentextraction.Format,
		len(derivations),
	)
	for _, derivation := range derivations {
		sourceFormatsByTargetFormat[derivation.TargetFormat] = append(
			sourceFormatsByTargetFormat[derivation.TargetFormat],
			derivation.SourceFormat,
		)
	}
	return derivationReach{sourceFormatsByTargetFormat: sourceFormatsByTargetFormat}
}

func (r derivationReach) ensureNoCycle() error {
	for targetFormat := range r.sourceFormatsByTargetFormat {
		if err := r.ensureNoCycleFrom(
			targetFormat, map[documentextraction.Format]bool{},
		); err != nil {
			return err
		}
	}
	return nil
}

func (r derivationReach) ensureNoCycleFrom(
	format documentextraction.Format,
	pendingFormats map[documentextraction.Format]bool,
) error {
	if pendingFormats[format] {
		return fmt.Errorf("%s derives from itself", format)
	}
	pendingFormats[format] = true
	defer delete(pendingFormats, format)
	for _, sourceFormat := range r.sourceFormatsByTargetFormat[format] {
		if err := r.ensureNoCycleFrom(sourceFormat, pendingFormats); err != nil {
			return err
		}
	}
	return nil
}

func (r derivationReach) ensureNoDanglingFormat(
	emittedFormats []documentextraction.Format,
) error {
	for _, targetFormat := range r.targetFormats() {
		if !r.reachedByAnyOf(targetFormat, emittedFormats) {
			return fmt.Errorf("no source format derives %s", targetFormat)
		}
	}
	for _, emittedFormat := range emittedFormats {
		if !r.reachesAnyTargetFormat(emittedFormat) {
			return fmt.Errorf("%s derives no target format", emittedFormat)
		}
	}
	return nil
}

func (r derivationReach) targetFormats() []documentextraction.Format {
	targetFormats := make(
		[]documentextraction.Format, 0, len(r.sourceFormatsByTargetFormat),
	)
	for targetFormat := range r.sourceFormatsByTargetFormat {
		targetFormats = append(targetFormats, targetFormat)
	}
	return targetFormats
}

func (r derivationReach) reachedByAnyOf(
	targetFormat documentextraction.Format,
	emittedFormats []documentextraction.Format,
) bool {
	formatsReaching := r.formatsReaching(targetFormat)
	for _, emittedFormat := range emittedFormats {
		if formatsReaching[emittedFormat] {
			return true
		}
	}
	return false
}

func (r derivationReach) formatsReaching(
	targetFormat documentextraction.Format,
) map[documentextraction.Format]bool {
	formatsReaching := map[documentextraction.Format]bool{targetFormat: true}
	unwalkedFormats := []documentextraction.Format{targetFormat}
	for len(unwalkedFormats) > 0 {
		format := unwalkedFormats[len(unwalkedFormats)-1]
		unwalkedFormats = unwalkedFormats[:len(unwalkedFormats)-1]
		for _, sourceFormat := range r.sourceFormatsByTargetFormat[format] {
			if formatsReaching[sourceFormat] {
				continue
			}
			formatsReaching[sourceFormat] = true
			unwalkedFormats = append(unwalkedFormats, sourceFormat)
		}
	}
	return formatsReaching
}

func (r derivationReach) reachesAnyTargetFormat(
	emittedFormat documentextraction.Format,
) bool {
	for _, targetFormat := range r.targetFormats() {
		if r.formatsReaching(targetFormat)[emittedFormat] {
			return true
		}
	}
	return false
}
