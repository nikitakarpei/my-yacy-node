package wordjoined

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type urlMetadataLookupRound struct {
	documentsWithoutMetadata distinctDocuments
	asks                     []peerasks.URLMetadataAsk
	answeredAsks             []peerasks.AnsweredURLMetadataAsk
	end                      URLMetadataLookupEnd
}

type URLMetadataLookupEnd string

const (
	URLMetadataLookupEndedByCoverage        URLMetadataLookupEnd = "coverage"
	URLMetadataLookupEndedByEveryAskSettled URLMetadataLookupEnd = "every ask settled"
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

func answeredAsksUntilCoverageFrom(
	outcomesAsTheySettle <-chan peerasks.URLMetadataAskOutcome,
	lookedUpDocuments distinctDocuments,
) ([]peerasks.AnsweredURLMetadataAsk, URLMetadataLookupEnd) {
	var answeredAsks []peerasks.AnsweredURLMetadataAsk
	documentsWithMetadata := distinctDocuments{}
	for outcome := range outcomesAsTheySettle {
		answeredAsk, answered := outcome.Answer.Get()
		if !answered {
			continue
		}
		answeredAsks = append(answeredAsks, answeredAsk)
		addLookedUpDocumentsOf(answeredAsk, lookedUpDocuments, documentsWithMetadata)
		if len(documentsWithMetadata) == len(lookedUpDocuments) {
			return answeredAsks, URLMetadataLookupEndedByCoverage
		}
	}

	return answeredAsks, URLMetadataLookupEndedByEveryAskSettled
}

func addLookedUpDocumentsOf(
	answeredAsk peerasks.AnsweredURLMetadataAsk,
	lookedUpDocuments distinctDocuments,
	documentsWithMetadata distinctDocuments,
) {
	for _, metadata := range answeredAsk.MetadataOfEachDocument {
		if !lookedUpDocuments.contains(metadata.Hash) {
			continue
		}
		documentsWithMetadata.add(metadata.Hash)
	}
}
