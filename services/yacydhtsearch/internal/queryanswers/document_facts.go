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

func FactsPerDocumentOf(
	postingReplicasPerDocument PostingReplicasPerDocument,
) FactsPerDocument {
	factsPerDocument := make(FactsPerDocument, len(postingReplicasPerDocument))
	for document, replicas := range postingReplicasPerDocument {
		facts, counted := factsOfTheFirstReplicaOfEachWord(replicas)
		if !counted {
			continue
		}
		factsPerDocument[document] = facts
	}

	return factsPerDocument
}

func factsOfTheFirstReplicaOfEachWord(replicas []PostingReplica) (DocumentFacts, bool) {
	facts := DocumentFacts{HitsPerQueryWord: map[yacymodel.Hash]int{}}
	counted := false
	for _, replica := range replicas {
		word, wordIsKnown := replica.Word.Get()
		if !wordIsKnown {
			continue
		}
		if _, alreadyCounted := facts.HitsPerQueryWord[word]; alreadyCounted {
			continue
		}
		facts.HitsPerQueryWord[word] = replica.Posting.Hits
		facts.AmountOfLinks = yacymodel.Some(
			replica.Posting.LocalLinks + replica.Posting.ExternalLinks,
		)
		counted = true
	}

	return facts, counted
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
