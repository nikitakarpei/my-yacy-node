package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
)

type findingsSender struct {
	findings chan<- queryfindings.Findings
	query    searchquery.Query
}

func findingsSenderFor(
	findings chan<- queryfindings.Findings,
	query searchquery.Query,
) findingsSender {
	return findingsSender{findings: findings, query: query}
}

func (sender findingsSender) sendFindingsAsEachAskSettles(
	settledAsksAsTheySettle <-chan wordpartitionasks.SettledAsk,
) []wordpartitionasks.SettledAsk {
	var settledAsks []wordpartitionasks.SettledAsk
	for settledAsk := range settledAsksAsTheySettle {
		settledAsks = append(settledAsks, settledAsk)
		sender.findings <- findingsFrom(settledAsks, sender.query)
	}

	return settledAsks
}
