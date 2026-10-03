package wordjoined

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type documentsPerQueryWord map[yacymodel.Hash]distinctDocuments

func (documentsOfEachQueryWord documentsPerQueryWord) documentsOfEveryQueryWord() distinctDocuments {
	documentsOfEveryQueryWord := distinctDocuments{}
	for _, documentsOfOneQueryWord := range documentsOfEachQueryWord {
		for document := range documentsOfOneQueryWord {
			if !documentsOfEachQueryWord.containForEveryQueryWord(document) {
				continue
			}
			documentsOfEveryQueryWord.add(document)
		}
	}

	return documentsOfEveryQueryWord
}

func (documentsOfEachQueryWord documentsPerQueryWord) containForEveryQueryWord(
	document yacymodel.URLHash,
) bool {
	for _, documentsOfOneQueryWord := range documentsOfEachQueryWord {
		if !documentsOfOneQueryWord.contains(document) {
			return false
		}
	}

	return true
}

func (documentsOfEachQueryWord documentsPerQueryWord) add(
	word yacymodel.Hash,
	documents distinctDocuments,
) {
	if documentsOfEachQueryWord[word] == nil {
		documentsOfEachQueryWord[word] = distinctDocuments{}
	}
	maps.Copy(documentsOfEachQueryWord[word], documents)
}
