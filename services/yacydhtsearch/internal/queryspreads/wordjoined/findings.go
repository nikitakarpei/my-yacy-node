package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func findingsOf(
	query searchquery.Query,
	foundDocuments []queryfindings.FoundDocument,
	documentsHeldPerQueryWord map[yacymodel.Hash]int,
) queryfindings.Findings {
	return queryfindings.Findings{
		QueryWords:                query.WordHashes(),
		CompoundWords:             query.CompoundWords,
		FoundDocuments:            foundDocuments,
		DocumentsHeldPerQueryWord: documentsHeldPerQueryWord,
	}
}
