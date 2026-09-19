package pagereading

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ReadPages struct {
	DocumentTextPerDocument map[yacymodel.URLHash]documenttext.DocumentText
	GoneDocuments           map[yacymodel.URLHash]struct{}
}

func readPagesFrom(pageReadResults []pageReadResult) ReadPages {
	readPages := ReadPages{
		DocumentTextPerDocument: make(
			map[yacymodel.URLHash]documenttext.DocumentText, len(pageReadResults),
		),
		GoneDocuments: map[yacymodel.URLHash]struct{}{},
	}
	for _, pageReadResult := range pageReadResults {
		switch pageReadResult.outcome {
		case pageWasRead:
			readPages.DocumentTextPerDocument[pageReadResult.document] = pageReadResult.text
		case pageWasGone:
			readPages.GoneDocuments[pageReadResult.document] = struct{}{}
		default:
		}
	}

	return readPages
}
