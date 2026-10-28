package documentjoin

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type listedDocumentsPerQueryWord map[yacymodel.Hash]yacymodel.URLHashes

func noListedDocumentsPerQueryWord(queryWords []yacymodel.Hash) listedDocumentsPerQueryWord {
	listedDocuments := make(listedDocumentsPerQueryWord, len(queryWords))
	for _, queryWord := range queryWords {
		listedDocuments[queryWord] = yacymodel.URLHashes{}
	}

	return listedDocuments
}

func (listedDocuments listedDocumentsPerQueryWord) add(
	word yacymodel.Hash,
	documents yacymodel.URLHashes,
) {
	listedDocuments[word] = yacymodel.UnionOf(listedDocuments[word], documents)
}

func (listedDocuments listedDocumentsPerQueryWord) listedForEveryQueryWord(
	document yacymodel.URLHash,
) bool {
	for _, documentsOfOneQueryWord := range listedDocuments {
		if !documentsOfOneQueryWord.Contains(document) {
			return false
		}
	}

	return true
}
