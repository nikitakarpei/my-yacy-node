package documentsearch

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type postingFilter struct {
	language           yacymodel.Optional[yacymodel.Language]
	requiredDocuments  map[yacymodel.URLHash]struct{}
	excludedDocuments  map[yacymodel.URLHash]struct{}
	siteHash           yacymodel.Optional[yacymodel.HostHash]
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

func (f postingFilter) matches(posting yacymodel.RWIPosting) bool {
	if wanted, ok := f.language.Get(); ok {
		code, present := posting.Language.Get()
		if !present || code != wanted {
			return false
		}
	}
	document := posting.URLHash
	if len(f.requiredDocuments) != 0 {
		if _, ok := f.requiredDocuments[document]; !ok {
			return false
		}
	}
	if _, ok := f.excludedDocuments[document]; ok {
		return false
	}
	if !matchesSiteHost(posting.URLHash, f.siteHash) {
		return false
	}
	if !matchesContentKind(posting, f.contentKind, f.strictContentKind) {
		return false
	}

	return matchesRequiredProperties(posting, f.requiredProperties)
}

func matchesSiteHost(
	location yacymodel.URLHash,
	siteHash yacymodel.Optional[yacymodel.HostHash],
) bool {
	wanted, ok := siteHash.Get()
	if !ok {
		return true
	}

	return location.HostHash() == wanted
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

func documentSet(identifiers []yacymodel.URLHash) map[yacymodel.URLHash]struct{} {
	if len(identifiers) == 0 {
		return nil
	}
	set := make(map[yacymodel.URLHash]struct{}, len(identifiers))
	for _, identifier := range identifiers {
		set[identifier] = struct{}{}
	}

	return set
}
