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
	f.HitsPerQueryWord = text.HitsPerQueryWord
	f.AmountOfWords = text.AmountOfWords
	f.QueryPhraseHits = text.QueryPhraseHits
	f.Snippet = text.Snippet

	return f
}

// TECHDEBT: naming — CountedByAPeer names a peer as the only counter, while the
// hits and the amount of words also come from the text of a document that was read.
func (f FoundDocument) CountedByAPeer() bool {
	if f.AmountOfWords > 0 {
		return true
	}
	for _, hits := range f.HitsPerQueryWord {
		if hits > 0 {
			return true
		}
	}

	return false
}
