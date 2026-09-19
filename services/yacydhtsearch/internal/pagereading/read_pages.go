package pagereading

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ReadPages struct {
	PageContentsPerDocument map[yacymodel.URLHash]queryanswers.PageContents
	GoneDocuments           map[yacymodel.URLHash]struct{}
}

func readPagesFrom(pageReadResults []pageReadResult) ReadPages {
	readPages := ReadPages{
		PageContentsPerDocument: make(
			map[yacymodel.URLHash]queryanswers.PageContents, len(pageReadResults),
		),
		GoneDocuments: map[yacymodel.URLHash]struct{}{},
	}
	for _, pageReadResult := range pageReadResults {
		switch pageReadResult.outcome {
		case pageWasRead:
			readPages.PageContentsPerDocument[pageReadResult.document] = pageReadResult.pageContents
		case pageWasGone:
			readPages.GoneDocuments[pageReadResult.document] = struct{}{}
		default:
		}
	}

	return readPages
}
