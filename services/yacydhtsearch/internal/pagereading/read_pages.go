package pagereading

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ReadPages struct {
	PageContentsPerDocument map[yacymodel.URLHash]pagecontents.PageContents
	WithdrawnDocuments      map[yacymodel.URLHash]struct{}
}

func readPagesFrom(pageReadResults []pageReadResult) ReadPages {
	readPages := ReadPages{
		PageContentsPerDocument: make(
			map[yacymodel.URLHash]pagecontents.PageContents, len(pageReadResults),
		),
		WithdrawnDocuments: map[yacymodel.URLHash]struct{}{},
	}
	for _, pageReadResult := range pageReadResults {
		switch pageReadResult.outcome {
		case pageWasRead:
			readPages.PageContentsPerDocument[pageReadResult.document] = pageReadResult.pageContents
		case pageWasGone, pageRefusesIndexing:
			readPages.WithdrawnDocuments[pageReadResult.document] = struct{}{}
		default:
		}
	}

	return readPages
}
