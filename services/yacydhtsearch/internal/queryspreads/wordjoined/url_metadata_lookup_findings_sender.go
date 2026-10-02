package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

type urlMetadataLookupFindingsSender struct {
	findings chan<- queryfindings.Findings
	query    searchquery.Query
	answers  *discoveryAnswers
}

func (sender urlMetadataLookupFindingsSender) sendFindingsOf(
	answeredAsks []peerasks.AnsweredURLMetadataAsk,
) {
	sender.findings <- findingsAfterURLMetadataLookup(sender.query, sender.answers, answeredAsks)
}
