package wordholdings

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type documentsPerQueryWord map[yacymodel.Hash]yacymodel.URLHashes

func (documentsOfEachQueryWord documentsPerQueryWord) documentsWithEveryQueryWord() yacymodel.URLHashes {
	documentsWithEveryQueryWord := yacymodel.URLHashes{}
	for _, documentsOfOneQueryWord := range documentsOfEachQueryWord {
		for document := range documentsOfOneQueryWord {
			if !documentsOfEachQueryWord.containForEveryQueryWord(document) {
				continue
			}
			documentsWithEveryQueryWord.Add(document)
		}
	}

	return documentsWithEveryQueryWord
}

func (documentsOfEachQueryWord documentsPerQueryWord) containForEveryQueryWord(
	document yacymodel.URLHash,
) bool {
	for _, documentsOfOneQueryWord := range documentsOfEachQueryWord {
		if !documentsOfOneQueryWord.Contains(document) {
			return false
		}
	}

	return true
}

func (documentsOfEachQueryWord documentsPerQueryWord) add(
	word yacymodel.Hash,
	documents yacymodel.URLHashes,
) {
	if documentsOfEachQueryWord[word] == nil {
		documentsOfEachQueryWord[word] = yacymodel.URLHashes{}
	}
	maps.Copy(documentsOfEachQueryWord[word], documents)
}
