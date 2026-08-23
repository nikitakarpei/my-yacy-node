package pageformats

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/derivationreach"
)

type DerivableFormats struct {
	byTargetFormat map[documentextraction.Format][]formatDerivation
}

func derivableFormatsOf(derivations []formatDerivation) (DerivableFormats, error) {
	derivationFormats := derivationFormatsOf(derivations)
	if err := derivationreach.EnsureNoCycle(derivationFormats); err != nil {
		return DerivableFormats{}, err
	}
	if err := derivationreach.EnsureNoDanglingFormat(
		derivationFormats, documentextraction.EmittedFormats(),
	); err != nil {
		return DerivableFormats{}, err
	}
	return DerivableFormats{
		byTargetFormat: derivationsByTargetFormat(derivations),
	}, nil
}

func derivationFormatsOf(
	derivations []formatDerivation,
) []derivationreach.DerivationFormats {
	derivationFormats := make(
		[]derivationreach.DerivationFormats, 0, len(derivations),
	)
	for _, derivation := range derivations {
		derivationFormats = append(derivationFormats, derivationreach.DerivationFormats{
			SourceFormat: derivation.SourceFormat(),
			TargetFormat: derivation.TargetFormat(),
		})
	}
	return derivationFormats
}

func derivationsByTargetFormat(
	derivations []formatDerivation,
) map[documentextraction.Format][]formatDerivation {
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
	return byTargetFormat
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
