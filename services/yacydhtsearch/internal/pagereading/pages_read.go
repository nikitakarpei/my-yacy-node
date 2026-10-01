package pagereading

import (
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PagesRead struct {
	PageContentsPerDocument map[yacymodel.URLHash]pagecontents.PageContents
	SpamVerdictPerDocument  map[yacymodel.URLHash]spamassessment.Verdict
	WithdrawnDocuments      map[yacymodel.URLHash]struct{}
}

func pagesReadFrom(pageReadResults []pageReadResult) PagesRead {
	pagesRead := PagesRead{
		PageContentsPerDocument: make(
			map[yacymodel.URLHash]pagecontents.PageContents, len(pageReadResults),
		),
		SpamVerdictPerDocument: make(
			map[yacymodel.URLHash]spamassessment.Verdict, len(pageReadResults),
		),
		WithdrawnDocuments: map[yacymodel.URLHash]struct{}{},
	}
	for _, pageReadResult := range pageReadResults {
		switch pageReadResult.outcome {
		case pageWasRead:
			pagesRead.PageContentsPerDocument[pageReadResult.document] = pageReadResult.pageContents
			pagesRead.SpamVerdictPerDocument[pageReadResult.document] = pageReadResult.spamVerdict
		case pageWasGone, pageRefusesIndexing:
			pagesRead.WithdrawnDocuments[pageReadResult.document] = struct{}{}
		default:
		}
	}

	return pagesRead
}
