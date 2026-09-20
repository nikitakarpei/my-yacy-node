package queryanswers

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type FoundDocument struct {
	Hash           yacymodel.URLHash
	Address        string
	Title          string
	Snippet        string
	PublishedAt    yacymodel.Optional[time.Time]
	FaviconAddress string
}

func FoundDocumentFrom(metadata yacymodel.URLMetadata) FoundDocument {
	return FoundDocument{
		Hash:           metadata.Hash,
		Address:        metadata.Address,
		Title:          metadata.Title,
		Snippet:        metadata.Snippet,
		PublishedAt:    publicationInstantOf(metadata),
		FaviconAddress: metadata.FaviconAddress,
	}
}

func publicationInstantOf(metadata yacymodel.URLMetadata) yacymodel.Optional[time.Time] {
	day, ok := metadata.Modified.Get()
	if !ok {
		day, ok = metadata.Loaded.Get()
	}
	if !ok {
		return yacymodel.None[time.Time]()
	}

	return yacymodel.Some(day.Time())
}

func (f FoundDocument) withTheContentsOfItsReadPage(
	pageContents pagecontents.PageContents,
) FoundDocument {
	if pageContents.Address != "" {
		f.Address = pageContents.Address
	}
	if pageContents.Title != "" {
		f.Title = pageContents.Title
	}
	f.Snippet = pageContents.Snippet

	return f
}
