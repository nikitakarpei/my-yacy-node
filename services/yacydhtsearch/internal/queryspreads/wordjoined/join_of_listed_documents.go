package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentjoin"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type joinOfListedDocuments struct {
	documentsJoiner *documentjoin.Joiner
}

func (join joinOfListedDocuments) WordPartitionAnswered(
	word yacymodel.Hash,
	_ uint,
	answers []wordpartitionasks.ReplicaAnswer,
) {
	join.documentsJoiner.ListUnder(word, documentsListedIn(answers))
}

func documentsListedIn(answers []wordpartitionasks.ReplicaAnswer) yacymodel.URLHashes {
	listedDocumentsOfEachAnswer := make([]yacymodel.URLHashes, 0, len(answers))
	for _, answer := range answers {
		listedDocumentsOfEachAnswer = append(listedDocumentsOfEachAnswer, answer.ListedDocuments)
	}

	return yacymodel.UnionOf(listedDocumentsOfEachAnswer...)
}
