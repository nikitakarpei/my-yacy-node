package pageformats

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/derivationreach"
)

const msgDerivationFailed = "format derivation failed, the next derivation is tried"

type FormatDerivationCatalog struct {
	byTargetFormat map[documentextraction.Format][]formatDerivation
}

func formatDerivationCatalogOf(
	derivations []formatDerivation,
) (FormatDerivationCatalog, error) {
	if err := derivationreach.EnsureFormatsDerivable(
		derivations, documentextraction.EmittedFormats(),
	); err != nil {
		return FormatDerivationCatalog{}, err
	}
	return FormatDerivationCatalog{
		byTargetFormat: derivationsByTargetFormat(derivations),
	}, nil
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

func (c FormatDerivationCatalog) BodyIn(
	ctx context.Context,
	format documentextraction.Format,
	document documentextraction.Document,
	pageURL canonicalurl.CanonicalURL,
) ([]byte, bool) {
	if format == document.Format {
		return document.Body, true
	}
	for _, derivation := range c.byTargetFormat[format] {
		sourceBody, sourceDerived := c.BodyIn(
			ctx, derivation.SourceFormat(), document, pageURL,
		)
		if !sourceDerived {
			continue
		}
		if body, derived := bodyDerivedBy(
			ctx, derivation, pageURL, sourceBody,
		); derived {
			return body, true
		}
	}
	return nil, false
}

func bodyDerivedBy(
	ctx context.Context,
	derivation formatDerivation,
	pageURL canonicalurl.CanonicalURL,
	sourceBody []byte,
) ([]byte, bool) {
	body, derived, err := derivation.BodyFrom(ctx, pageURL, sourceBody)
	if err != nil {
		slog.WarnContext(ctx, msgDerivationFailed,
			slog.String("sourceFormat", string(derivation.SourceFormat())),
			slog.String("targetFormat", string(derivation.TargetFormat())),
			slog.Any("error", err),
		)
		return nil, false
	}
	return body, derived
}
