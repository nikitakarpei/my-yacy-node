package documentsperword

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type documentsCountingForQueryWord map[yacymodel.Hash]yacymodel.URLHashes

func (documentsOfEachQueryWord documentsCountingForQueryWord) documentsWithEveryQueryWord() yacymodel.URLHashes {
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

func (documentsOfEachQueryWord documentsCountingForQueryWord) containForEveryQueryWord(
	document yacymodel.URLHash,
) bool {
	for _, documentsOfOneQueryWord := range documentsOfEachQueryWord {
		if !documentsOfOneQueryWord.Contains(document) {
			return false
		}
	}

	return true
}

func (documentsOfEachQueryWord documentsCountingForQueryWord) add(
	word yacymodel.Hash,
	documents yacymodel.URLHashes,
) {
	if documentsOfEachQueryWord[word] == nil {
		documentsOfEachQueryWord[word] = yacymodel.URLHashes{}
	}
	maps.Copy(documentsOfEachQueryWord[word], documents)
}
