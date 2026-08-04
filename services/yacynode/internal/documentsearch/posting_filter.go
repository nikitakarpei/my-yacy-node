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
	requiredAppearance yacymodel.Optional[yacymodel.Appearance]
}

func (s searcher) postingFilter(
	ctx context.Context,
	criteria searchCriteria,
	excludedTerms []yacymodel.Hash,
) (postingFilter, error) {
	excludedDocuments, err := s.excludedDocuments(ctx, excludedTerms)
	if err != nil {
		return postingFilter{}, err
	}

	return postingFilter{
		language:           criteria.language,
		requiredDocuments:  documentSet(criteria.requiredDocuments),
		excludedDocuments:  excludedDocuments,
		siteHash:           criteria.siteHash,
		contentKind:        criteria.contentKind,
		strictContentKind:  criteria.strictContentKind,
		requiredAppearance: criteria.requiredAppearance,
	}, nil
}

func (f postingFilter) accepts(posting yacymodel.RWIPosting) bool {
	if requiredLanguage, ok := f.language.Get(); ok {
		code, ok := posting.Language.Get()
		if !ok || code != requiredLanguage {
			return false
		}
	}
	documentHash := posting.URLHash
	if len(f.requiredDocuments) != 0 {
		if _, ok := f.requiredDocuments[documentHash]; !ok {
			return false
		}
	}
	if _, ok := f.excludedDocuments[documentHash]; ok {
		return false
	}
	if !isFromRequestedSite(posting.URLHash, f.siteHash) {
		return false
	}
	if !matchesContentKind(posting, f.contentKind, f.strictContentKind) {
		return false
	}

	return sharesRequiredAppearance(posting, f.requiredAppearance)
}

func isFromRequestedSite(
	documentHash yacymodel.URLHash,
	requestedSite yacymodel.Optional[yacymodel.HostHash],
) bool {
	wanted, ok := requestedSite.Get()
	if !ok {
		return true
	}

	return documentHash.HostHash() == wanted
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

func sharesRequiredAppearance(
	posting yacymodel.RWIPosting,
	requiredAppearance yacymodel.Optional[yacymodel.Appearance],
) bool {
	traits, ok := requiredAppearance.Get()
	if !ok {
		return true
	}

	return posting.Appearance.OverlapsAny(traits)
}

func documentSet(documentHashes []yacymodel.URLHash) map[yacymodel.URLHash]struct{} {
	if len(documentHashes) == 0 {
		return nil
	}
	set := make(map[yacymodel.URLHash]struct{}, len(documentHashes))
	for _, documentHash := range documentHashes {
		set[documentHash] = struct{}{}
	}

	return set
}
