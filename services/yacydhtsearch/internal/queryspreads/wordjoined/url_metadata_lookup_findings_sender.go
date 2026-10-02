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
	sender.findings <- sender.findingsFrom(answeredAsks)
}

func (sender urlMetadataLookupFindingsSender) findingsFrom(
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
