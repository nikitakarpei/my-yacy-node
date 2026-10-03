package documentsperword

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type documentsOfEachQueryWord map[yacymodel.Hash]yacymodel.URLHashes

func (documentsOfWords documentsOfEachQueryWord) documentsWithEveryQueryWord() yacymodel.URLHashes {
	documentsWithEveryQueryWord := yacymodel.URLHashes{}
	for _, documentsOfOneQueryWord := range documentsOfWords {
		for document := range documentsOfOneQueryWord {
			if !documentsOfWords.containForEveryQueryWord(document) {
				continue
			}
			documentsWithEveryQueryWord.Add(document)
		}
	}

	return documentsWithEveryQueryWord
}

func (documentsOfWords documentsOfEachQueryWord) containForEveryQueryWord(
	document yacymodel.URLHash,
) bool {
	for _, documentsOfOneQueryWord := range documentsOfWords {
		if !documentsOfOneQueryWord.Contains(document) {
			return false
		}
	}

	return true
}

func (documentsOfWords documentsOfEachQueryWord) add(
	word yacymodel.Hash,
	documents yacymodel.URLHashes,
) {
	if documentsOfWords[word] == nil {
		documentsOfWords[word] = yacymodel.URLHashes{}
	}
	maps.Copy(documentsOfWords[word], documents)
}
