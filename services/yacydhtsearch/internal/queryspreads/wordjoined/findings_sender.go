package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type discoveryFindingsSender struct {
	findings   chan<- queryfindings.Findings
	query      searchquery.Query
	partitions yacymodel.DHTRingPartitions
}

func discoveryFindingsSenderFor(
	findings chan<- queryfindings.Findings,
	query searchquery.Query,
	partitions yacymodel.DHTRingPartitions,
) discoveryFindingsSender {
	return discoveryFindingsSender{findings: findings, query: query, partitions: partitions}
}

func (sender discoveryFindingsSender) sendFindingsOf(settledAsks settledAsks) {
	discoveryRound := discoveryRoundSoFarFrom(settledAsks, sender.query, sender.partitions)
	sender.findings <- findingsFrom(
		sender.query, discoveryRound, discoveryRound.joinedDocuments(), nil,
	)
}

func discoveryRoundSoFarFrom(
	settledAsks settledAsks,
	query searchquery.Query,
	partitions yacymodel.DHTRingPartitions,
) discoveryRound {
	return discoveryRound{
		queryWords:  query.WordHashes(),
		settledAsks: settledAsks,
		queryWordsFewestDocumentsFirst: queryWordsFewestDocumentsFirstFrom(
			query.WordHashes(), settledAsks, partitions,
		),
		compoundWords: compoundWordsAcrossReplicasFrom(
			query.CompoundWords, settledAsks, partitions,
		),
	}
}

func (sender discoveryFindingsSender) lookupFindingsSenderAfter(
	discoveryRound discoveryRound,
	joinedDocuments distinctDocuments,
) lookupFindingsSender {
	return lookupFindingsSender{
		findings:        sender.findings,
		query:           sender.query,
		discoveryRound:  discoveryRound,
		joinedDocuments: joinedDocuments,
	}
}

type lookupFindingsSender struct {
	findings        chan<- queryfindings.Findings
	query           searchquery.Query
	discoveryRound  discoveryRound
	joinedDocuments distinctDocuments
}

func (sender lookupFindingsSender) sendFindingsOf(
	answeredAsks []peerasks.AnsweredURLMetadataAsk,
) {
	sender.findings <- findingsFrom(
		sender.query, sender.discoveryRound, sender.joinedDocuments, answeredAsks,
	)
}
