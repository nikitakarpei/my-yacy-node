package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

type discoveryFindingsSender struct {
	findings chan<- queryfindings.Findings
	query    searchquery.Query
}

func (sender discoveryFindingsSender) sendFindingsOf(answers *discoveryAnswers) {
	sender.findings <- findingsOf(
		sender.query, answers.foundDocuments(), answers.amountOfDocumentsHeldPerQueryWord(),
	)
}
