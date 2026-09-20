package queryanswers

import (
	"maps"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type FoundDocument struct {
	Hash                         yacymodel.URLHash
	Address                      string
	Title                        string
	Snippet                      string
	PublishedAt                  yacymodel.Optional[time.Time]
	FaviconAddress               string
	HitsPerQueryWord             map[yacymodel.Hash]int
	AmountOfWordsAPeerCounted    yacymodel.Optional[int]
	AmountOfWordsOfTheReadPage   yacymodel.Optional[int]
	QueryPhraseHitsOfTheReadPage yacymodel.Optional[int]
	LinkCounts                   yacymodel.Optional[pagecontents.LinkCounts]
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

func (f FoundDocument) AmountOfWordsAnyoneCounted() yacymodel.Optional[int] {
	if f.AmountOfWordsOfTheReadPage.Present() {
		return f.AmountOfWordsOfTheReadPage
	}

	return f.AmountOfWordsAPeerCounted
}

func (f FoundDocument) saturatedWith(pageContents pagecontents.PageContents) FoundDocument {
	if pageContents.Address != "" {
		f.Address = pageContents.Address
	}
	if pageContents.Title != "" {
		f.Title = pageContents.Title
	}
	f.HitsPerQueryWord = f.hitsPerQueryWordSaturatedWith(pageContents.HitsPerQueryWord)
	f.AmountOfWordsOfTheReadPage = yacymodel.Some(pageContents.AmountOfWords)
	f.QueryPhraseHitsOfTheReadPage = yacymodel.Some(pageContents.QueryPhraseHits)
	f.Snippet = pageContents.Snippet
	f.LinkCounts = yacymodel.Some(pageContents.LinkCounts)

	return f
}

func (f FoundDocument) hitsPerQueryWordSaturatedWith(
	hitsPerQueryWordOfTheReadPage map[yacymodel.Hash]int,
) map[yacymodel.Hash]int {
	hitsPerQueryWord := make(map[yacymodel.Hash]int, len(hitsPerQueryWordOfTheReadPage))
	maps.Copy(hitsPerQueryWord, hitsPerQueryWordOfTheReadPage)
	for word, hitsAPeerCounted := range f.HitsPerQueryWord {
		if hitsPerQueryWord[word] > 0 {
			continue
		}
		hitsPerQueryWord[word] = hitsAPeerCounted
	}

	return hitsPerQueryWord
}
