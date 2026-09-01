// Package postingfilter decides which postings of a term the search criteria
// admit. It reads no storage: the caller supplies the documents that hold an
// excluded term.
package postingfilter

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
)

type Filter struct {
	language           yacymodel.Optional[yacymodel.Language]
	requiredDocuments  map[yacymodel.URLHash]struct{}
	excludedDocuments  map[yacymodel.URLHash]struct{}
	siteHash           yacymodel.Optional[yacymodel.HostHash]
	contentKind        searchcriteria.ContentKind
	strictContentKind  bool
	requiredAppearance yacymodel.Optional[yacymodel.Appearance]
}

func FilterForSearch(
	criteria searchcriteria.Criteria,
	excludedDocuments map[yacymodel.URLHash]struct{},
) Filter {
	return Filter{
		language:           criteria.Language,
		requiredDocuments:  documentSet(criteria.RequiredDocuments),
		excludedDocuments:  excludedDocuments,
		siteHash:           criteria.SiteHash,
		contentKind:        criteria.ContentKind,
		strictContentKind:  criteria.StrictContentKind,
		requiredAppearance: criteria.RequiredAppearance,
	}
}

func FilterForReport(criteria searchcriteria.Criteria) Filter {
	return Filter{
		language:           criteria.Language,
		requiredDocuments:  documentSet(criteria.RequiredDocuments),
		siteHash:           criteria.SiteHash,
		contentKind:        criteria.ContentKind,
		strictContentKind:  criteria.StrictContentKind,
		requiredAppearance: criteria.RequiredAppearance,
	}
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

func (f Filter) Accepts(posting yacymodel.RWIPosting) bool {
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

func isOfDocumentType(posting yacymodel.RWIPosting, kind searchcriteria.ContentKind) bool {
	switch kind {
	case searchcriteria.ImageContent:
		return posting.DocumentType == yacymodel.DocumentTypeImage
	case searchcriteria.AudioContent:
		return posting.DocumentType == yacymodel.DocumentTypeAudio
	case searchcriteria.VideoContent:
		return posting.DocumentType == yacymodel.DocumentTypeMovie
	case searchcriteria.ApplicationContent:
		return posting.Appearance.HasApp
	default:
		return true
	}
}

func appearsAsContentKind(posting yacymodel.RWIPosting, kind searchcriteria.ContentKind) bool {
	switch kind {
	case searchcriteria.ImageContent:
		return posting.Appearance.HasImage
	case searchcriteria.AudioContent:
		return posting.Appearance.HasAudio
	case searchcriteria.VideoContent:
		return posting.Appearance.HasVideo
	case searchcriteria.ApplicationContent:
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
