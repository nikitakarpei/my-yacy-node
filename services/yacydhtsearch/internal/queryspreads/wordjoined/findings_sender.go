package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

type discoveryFindingsSender struct {
	findings chan<- queryfindings.Findings
	query    searchquery.Query
}

func discoveryFindingsSenderFor(
	findings chan<- queryfindings.Findings,
	query searchquery.Query,
) discoveryFindingsSender {
	return discoveryFindingsSender{findings: findings, query: query}
}

func (sender discoveryFindingsSender) sendFindingsOf(answers *discoveryAnswers) {
	sender.findings <- findingsOf(
		sender.query, answers.foundDocuments(), answers.amountOfDocumentsHeldPerQueryWord(),
	)
}

func (sender discoveryFindingsSender) lookupFindingsSenderAfter(
	answers *discoveryAnswers,
) lookupFindingsSender {
	return lookupFindingsSender{findings: sender.findings, query: sender.query, answers: answers}
}

type lookupFindingsSender struct {
	findings chan<- queryfindings.Findings
	query    searchquery.Query
	answers  *discoveryAnswers
}

func (sender lookupFindingsSender) sendFindingsOf(
	answeredAsks []peerasks.AnsweredURLMetadataAsk,
) {
	sender.findings <- sender.findingsFrom(answeredAsks)
}

func (sender lookupFindingsSender) findingsFrom(
	answeredAsks []peerasks.AnsweredURLMetadataAsk,
) queryfindings.Findings {
	documentsThePeersSent := sender.answers.joinedDocumentsThePeersSent()
	for _, answeredAsk := range answeredAsks {
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			documentsThePeersSent.KeepMetadataThePeerSent(metadata, answeredAsk.Ask.Peer.Hash)
		}
	}

	return findingsOf(
		sender.query,
		documentsThePeersSent.FoundDocuments(),
		sender.answers.amountOfDocumentsHeldPerQueryWord(),
	)
}
