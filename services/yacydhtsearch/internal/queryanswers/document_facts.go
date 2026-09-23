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

func documentFactsOfFirstReplicaOfEachWord(replicas []PostingReplica) DocumentFacts {
	hitsPerQueryWord := map[yacymodel.Hash]int{}
	amountOfLinks := yacymodel.None[int]()
	for _, replica := range replicas {
		if _, alreadyCounted := hitsPerQueryWord[replica.Word]; alreadyCounted {
			continue
		}
		hitsPerQueryWord[replica.Word] = replica.Posting.Hits
		amountOfLinks = yacymodel.Some(replica.Posting.LocalLinks + replica.Posting.ExternalLinks)
	}
	if len(hitsPerQueryWord) == 0 {
		return DocumentFacts{}
	}

	return DocumentFacts{HitsPerQueryWord: hitsPerQueryWord, AmountOfLinks: amountOfLinks}
}

func documentFactsOfReadPage(pageContents pagecontents.PageContents) DocumentFacts {
	return DocumentFacts{
		HitsPerQueryWord: pageContents.HitsPerQueryWord,
		QueryPhraseHits:  yacymodel.Some(pageContents.QueryPhraseHits),
		AmountOfWords:    yacymodel.Some(pageContents.AmountOfWords),
		AmountOfLinks:    yacymodel.Some(pageContents.LinkCounts.AmountOfLinks()),
	}
}
