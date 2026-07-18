package documentsearch

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const msgSiteHostUndetermined = "rwi search posting site host undetermined"

type postingFilter struct {
	language           string
	requiredDocuments  map[yacymodel.Hash]struct{}
	excludedDocuments  map[yacymodel.Hash]struct{}
	siteHash           string
	contentKind        contentKind
	strictContentKind  bool
	requiredProperties yacymodel.Optional[yacymodel.Appearance]
}

func (s searcher) postingFilter(
	ctx context.Context,
	criteria searchCriteria,
	excludedTerms []yacymodel.Hash,
) (postingFilter, error) {
	excluded, err := s.excludedDocuments(ctx, excludedTerms)
	if err != nil {
		return postingFilter{}, err
	}

	return postingFilter{
		language:           criteria.language,
		requiredDocuments:  documentSet(criteria.requiredDocuments),
		excludedDocuments:  excluded,
		siteHash:           criteria.siteHash,
		contentKind:        criteria.contentKind,
		strictContentKind:  criteria.strictContentKind,
		requiredProperties: criteria.requiredProperties,
	}, nil
}

func (f postingFilter) matches(ctx context.Context, posting yacymodel.RWIPosting) bool {
	if f.language != "" && string(posting.Language) != f.language {
		return false
	}
	document := posting.URLHash.Hash()
	if len(f.requiredDocuments) != 0 {
		if _, ok := f.requiredDocuments[document]; !ok {
			return false
		}
	}
	if _, ok := f.excludedDocuments[document]; ok {
		return false
	}
	if !matchesSiteHost(ctx, posting.URLHash, f.siteHash) {
		return false
	}
	if !matchesContentKind(posting, f.contentKind, f.strictContentKind) {
		return false
	}

	return matchesRequiredProperties(posting, f.requiredProperties)
}

func matchesSiteHost(ctx context.Context, location yacymodel.URLHash, siteHash string) bool {
	if siteHash == "" {
		return true
	}
	hostHash, err := location.HostHash()
	if err != nil {
		slog.WarnContext(ctx, msgSiteHostUndetermined, slog.Any("error", err))

		return false
	}

	return hostHash == siteHash
}

func matchesContentKind(posting yacymodel.RWIPosting, kind contentKind, strict bool) bool {
	switch kind {
	case imageContent:
		if strict {
			return posting.DocumentType == yacymodel.DocumentTypeImage
		}

		return posting.Appearance.HasImage
	case audioContent:
		if strict {
			return posting.DocumentType == yacymodel.DocumentTypeAudio
		}

		return posting.Appearance.HasAudio
	case videoContent:
		if strict {
			return posting.DocumentType == yacymodel.DocumentTypeMovie
		}

		return posting.Appearance.HasVideo
	case applicationContent:
		return posting.Appearance.HasApp
	default:
		return true
	}
}

func matchesRequiredProperties(
	posting yacymodel.RWIPosting,
	required yacymodel.Optional[yacymodel.Appearance],
) bool {
	traits, ok := required.Get()
	if !ok {
		return true
	}

	return posting.Appearance.OverlapsAny(traits)
}

func documentSet(identifiers []yacymodel.Hash) map[yacymodel.Hash]struct{} {
	if len(identifiers) == 0 {
		return nil
	}
	set := make(map[yacymodel.Hash]struct{}, len(identifiers))
	for _, identifier := range identifiers {
		set[identifier] = struct{}{}
	}

	return set
}
