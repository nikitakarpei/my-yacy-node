package queryanswers

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type FoundDocument struct {
	Hash             yacymodel.URLHash
	Address          string
	Title            string
	Snippet          string
	PublishedAt      yacymodel.Optional[time.Time]
	FaviconAddress   string
	HitsPerQueryWord map[yacymodel.Hash]int
	AmountOfWords    int
	QueryPhraseHits  int
}

func FoundDocumentFrom(metadata yacymodel.URLMetadata) FoundDocument {
	return FoundDocument{
		Hash:             metadata.Hash,
		Address:          metadata.Address,
		Title:            metadata.Title,
		Snippet:          metadata.Snippet,
		PublishedAt:      publicationInstantOf(metadata),
		FaviconAddress:   metadata.FaviconAddress,
		HitsPerQueryWord: map[yacymodel.Hash]int{},
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

func (f FoundDocument) saturatedWith(text documenttext.DocumentText) FoundDocument {
	if text.Address != "" {
		f.Address = text.Address
	}
	if text.Title != "" {
		f.Title = text.Title
	}
	f.HitsPerQueryWord = text.HitsPerQueryWord
	f.AmountOfWords = text.AmountOfWords
	f.QueryPhraseHits = text.QueryPhraseHits
	f.Snippet = text.Snippet

	return f
}
