package queryanswers

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type DocumentFacts struct {
	HitsPerQueryWord map[yacymodel.Hash]int
	QueryPhraseHits  yacymodel.Optional[int]
	AmountOfWords    yacymodel.Optional[int]
	AmountOfLinks    yacymodel.Optional[int]
}

type FactsPerDocument map[yacymodel.URLHash]DocumentFacts

func (f FactsPerDocument) KeepTheFirstPostingOfTheWord(
	document yacymodel.URLHash,
	word yacymodel.Hash,
	posting yacymodel.Optional[yacymodel.RWIPosting],
) {
	sentPosting, sent := posting.Get()
	if !sent {
		return
	}
	facts := f.factsCountedForTheDocument(document)
	if _, alreadyCounted := facts.HitsPerQueryWord[word]; alreadyCounted {
		return
	}
	facts.HitsPerQueryWord[word] = sentPosting.Hits
	facts.AmountOfLinks = yacymodel.Some(sentPosting.LocalLinks + sentPosting.ExternalLinks)
	f[document] = facts
}

func (f FactsPerDocument) factsCountedForTheDocument(
	document yacymodel.URLHash,
) DocumentFacts {
	facts, counted := f[document]
	if !counted {
		return DocumentFacts{HitsPerQueryWord: map[yacymodel.Hash]int{}}
	}

	return facts
}

func (f FactsPerDocument) withTheFactsOfTheReadPages(
	pageContentsPerDocument map[yacymodel.URLHash]pagecontents.PageContents,
) FactsPerDocument {
	factsPerDocument := make(FactsPerDocument, len(f))
	for document, facts := range f {
		factsPerDocument[document] = facts
	}
	for document, pageContents := range pageContentsPerDocument {
		factsPerDocument[document] = factsOfAReadPage(pageContents)
	}

	return factsPerDocument
}

func factsOfAReadPage(pageContents pagecontents.PageContents) DocumentFacts {
	return DocumentFacts{
		HitsPerQueryWord: pageContents.HitsPerQueryWord,
		QueryPhraseHits:  yacymodel.Some(pageContents.QueryPhraseHits),
		AmountOfWords:    yacymodel.Some(pageContents.AmountOfWords),
		AmountOfLinks:    yacymodel.Some(pageContents.LinkCounts.AmountOfLinks()),
	}
}

func (f FactsPerDocument) withoutDocuments(
	documents map[yacymodel.URLHash]struct{},
) FactsPerDocument {
	factsPerDocument := make(FactsPerDocument, len(f))
	for document, facts := range f {
		if _, left := documents[document]; left {
			continue
		}
		factsPerDocument[document] = facts
	}

	return factsPerDocument
}
