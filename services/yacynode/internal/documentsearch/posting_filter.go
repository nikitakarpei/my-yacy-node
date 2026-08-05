package documentsearch

import (
	"context"
	"fmt"

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

func (s searcher) searchPostingFilterFrom(
	ctx context.Context,
	criteria searchCriteria,
) (postingFilter, error) {
	excludedDocuments, err := s.documentsContaining(ctx, criteria.excludedTerms)
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

func (s searcher) documentsContaining(
	ctx context.Context,
	terms []yacymodel.Hash,
) (map[yacymodel.URLHash]struct{}, error) {
	documents := make(map[yacymodel.URLHash]struct{})
	for _, term := range terms {
		err := s.index.ScanWord(
			ctx,
			term,
			func(posting yacymodel.RWIPosting) (bool, error) {
				documents[posting.URLHash] = struct{}{}

				return true, nil
			},
		)
		if err != nil {
			return nil, fmt.Errorf("scan term: %w", err)
		}
	}

	return documents, nil
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

func reportPostingFilterFrom(criteria searchCriteria) postingFilter {
	return postingFilter{
		language:           criteria.language,
		requiredDocuments:  documentSet(criteria.requiredDocuments),
		siteHash:           criteria.siteHash,
		contentKind:        criteria.contentKind,
		strictContentKind:  criteria.strictContentKind,
		requiredAppearance: criteria.requiredAppearance,
	}
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
	if f.strictContentKind && !isOfDocumentType(posting, f.contentKind) {
		return false
	}
	if !f.strictContentKind && !appearsAsContentKind(posting, f.contentKind) {
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

func isOfDocumentType(posting yacymodel.RWIPosting, kind contentKind) bool {
	switch kind {
	case imageContent:
		return posting.DocumentType == yacymodel.DocumentTypeImage
	case audioContent:
		return posting.DocumentType == yacymodel.DocumentTypeAudio
	case videoContent:
		return posting.DocumentType == yacymodel.DocumentTypeMovie
	case applicationContent:
		return posting.Appearance.HasApp
	default:
		return true
	}
}

func appearsAsContentKind(posting yacymodel.RWIPosting, kind contentKind) bool {
	switch kind {
	case imageContent:
		return posting.Appearance.HasImage
	case audioContent:
		return posting.Appearance.HasAudio
	case videoContent:
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
