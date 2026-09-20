package queryanswers

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PostingsOfOneDocumentAcrossReplicas struct {
	hitsAcrossReplicasPerQueryWord map[yacymodel.Hash][]int
	amountsOfWordsAcrossReplicas   []int
	localLinksAcrossReplicas       []int
	externalLinksAcrossReplicas    []int
}

func (postings *PostingsOfOneDocumentAcrossReplicas) Take(
	word yacymodel.Hash,
	posting yacymodel.Optional[yacymodel.RWIPosting],
) {
	sentPosting, sent := posting.Get()
	if !sent {
		return
	}
	if postings.hitsAcrossReplicasPerQueryWord == nil {
		postings.hitsAcrossReplicasPerQueryWord = map[yacymodel.Hash][]int{}
	}
	postings.hitsAcrossReplicasPerQueryWord[word] = append(
		postings.hitsAcrossReplicasPerQueryWord[word], sentPosting.Hits,
	)
	postings.amountsOfWordsAcrossReplicas = append(
		postings.amountsOfWordsAcrossReplicas, sentPosting.TextWords,
	)
	postings.localLinksAcrossReplicas = append(
		postings.localLinksAcrossReplicas, sentPosting.LocalLinks,
	)
	postings.externalLinksAcrossReplicas = append(
		postings.externalLinksAcrossReplicas, sentPosting.ExternalLinks,
	)
}

func (postings PostingsOfOneDocumentAcrossReplicas) HitsPerQueryWord() map[yacymodel.Hash]int {
	hitsPerQueryWord := make(
		map[yacymodel.Hash]int, len(postings.hitsAcrossReplicasPerQueryWord),
	)
	for word, hitsAcrossReplicas := range postings.hitsAcrossReplicasPerQueryWord {
		hitsPerQueryWord[word] = lowerMedianOf(hitsAcrossReplicas)
	}

	return hitsPerQueryWord
}

func (postings PostingsOfOneDocumentAcrossReplicas) AmountOfWords() int {
	if len(postings.amountsOfWordsAcrossReplicas) == 0 {
		return 0
	}

	return lowerMedianOf(postings.amountsOfWordsAcrossReplicas)
}

func (postings PostingsOfOneDocumentAcrossReplicas) LinkCounts() yacymodel.Optional[pagecontents.LinkCounts] {
	if len(postings.localLinksAcrossReplicas) == 0 {
		return yacymodel.None[pagecontents.LinkCounts]()
	}

	return yacymodel.Some(pagecontents.LinkCounts{
		LocalLinks:    lowerMedianOf(postings.localLinksAcrossReplicas),
		ExternalLinks: lowerMedianOf(postings.externalLinksAcrossReplicas),
	})
}

func lowerMedianOf(amounts []int) int {
	sortedAmounts := slices.Sorted(slices.Values(amounts))

	return sortedAmounts[(len(sortedAmounts)-1)/2]
}
