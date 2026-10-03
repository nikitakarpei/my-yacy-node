package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type distinctDocuments map[yacymodel.URLHash]struct{}

func (documents distinctDocuments) contains(document yacymodel.URLHash) bool {
	_, contained := documents[document]

	return contained
}

func (documents distinctDocuments) add(document yacymodel.URLHash) {
	documents[document] = struct{}{}
}

func (documents distinctDocuments) addEach(addedDocuments []yacymodel.URLHash) {
	for _, document := range addedDocuments {
		documents.add(document)
	}
}
