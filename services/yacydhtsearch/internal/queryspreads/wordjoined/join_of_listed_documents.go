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
	listedDocuments := yacymodel.URLHashes{}
	for _, answer := range answers {
		for _, listedDocument := range answer.ListedDocuments {
			listedDocuments.Add(listedDocument.Hash)
		}
	}

	return listedDocuments
}
