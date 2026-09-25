package wordjoined

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type urlMetadataLookupRound struct {
	documentsWithoutMetadata distinctDocuments
	asks                     []peerasks.URLMetadataAsk
	answeredAsks             []peerasks.AnsweredURLMetadataAsk
	endReason                URLMetadataLookupEndReason
}

type URLMetadataLookupEndReason string

const (
	URLMetadataLookupEndedByCoverage        URLMetadataLookupEndReason = "coverage"
	URLMetadataLookupEndedByEveryAskSettled URLMetadataLookupEndReason = "every ask settled"
)

func documentsWithoutMetadataAmong(
	joinedDocuments distinctDocuments,
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
) distinctDocuments {
	documentsWithoutMetadata := maps.Clone(joinedDocuments)
	for _, answeredAsk := range answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			delete(documentsWithoutMetadata, matchedDocument.Metadata.Hash)
		}
	}

	return documentsWithoutMetadata
}

type urlMetadataLookupInFlight struct {
	lookedUpDocuments             distinctDocuments
	lookedUpDocumentsWithMetadata distinctDocuments
	answeredAsks                  []peerasks.AnsweredURLMetadataAsk
}

func urlMetadataLookupInFlightOf(asks []peerasks.URLMetadataAsk) urlMetadataLookupInFlight {
	return urlMetadataLookupInFlight{
		lookedUpDocuments:             lookedUpDocumentsAcross(asks),
		lookedUpDocumentsWithMetadata: distinctDocuments{},
	}
}

func lookedUpDocumentsAcross(
	asks []peerasks.URLMetadataAsk,
) distinctDocuments {
	documents := distinctDocuments{}
	for _, ask := range asks {
		for _, document := range ask.Documents {
			documents.add(document)
		}
	}

	return documents
}

func (lookup *urlMetadataLookupInFlight) settleUntilCoverage(
	outcomesAsTheySettle <-chan peerasks.URLMetadataAskOutcome,
) ([]peerasks.AnsweredURLMetadataAsk, URLMetadataLookupEndReason) {
	for outcome := range outcomesAsTheySettle {
		lookup.settle(outcome)
		if lookup.covered() {
			return lookup.answeredAsks, URLMetadataLookupEndedByCoverage
		}
	}

	return lookup.answeredAsks, URLMetadataLookupEndedByEveryAskSettled
}

func (lookup *urlMetadataLookupInFlight) settle(outcome peerasks.URLMetadataAskOutcome) {
	answeredAsk, answered := outcome.Answer.Get()
	if !answered {
		return
	}
	lookup.answeredAsks = append(lookup.answeredAsks, answeredAsk)
	lookup.lookedUpDocumentsWithMetadata.addEach(
		lookedUpDocumentsWithMetadataIn(answeredAsk, lookup.lookedUpDocuments),
	)
}

func lookedUpDocumentsWithMetadataIn(
	answeredAsk peerasks.AnsweredURLMetadataAsk,
	lookedUpDocuments distinctDocuments,
) []yacymodel.URLHash {
	var documentsWithMetadata []yacymodel.URLHash
	for _, metadata := range answeredAsk.MetadataOfEachDocument {
		if lookedUpDocuments.contains(metadata.Hash) {
			documentsWithMetadata = append(documentsWithMetadata, metadata.Hash)
		}
	}

	return documentsWithMetadata
}

func (lookup *urlMetadataLookupInFlight) covered() bool {
	return len(lookup.lookedUpDocumentsWithMetadata) == len(lookup.lookedUpDocuments)
}
