package wordjoined

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type urlMetadataLookupRound struct {
	documentsWithoutMetadata distinctDocuments
	asks                     []peerasks.URLMetadataAsk
	endedURLMetadataLookup
}

type endedURLMetadataLookup struct {
	answeredAsks                    []peerasks.AnsweredURLMetadataAsk
	end                             URLMetadataLookupEnd
	amountOfLookedUpDocumentsCutOff int
}

type URLMetadataLookupEnd string

const (
	URLMetadataLookupEndedByCoverage        URLMetadataLookupEnd = "coverage"
	URLMetadataLookupEndedByEveryAskSettled URLMetadataLookupEnd = "every ask settled"
	URLMetadataLookupEndedByCutoff          URLMetadataLookupEnd = "cut off"
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
	amountOfLookedUpDocuments     int
	asksInFlightPerOpenDocument   map[yacymodel.URLHash]int
	lookedUpDocumentsWithMetadata distinctDocuments
	answeredAsks                  []peerasks.AnsweredURLMetadataAsk
}

func urlMetadataLookupInFlightOf(asks []peerasks.URLMetadataAsk) urlMetadataLookupInFlight {
	asksInFlightPerOpenDocument := map[yacymodel.URLHash]int{}
	for _, ask := range asks {
		for _, document := range ask.Documents {
			asksInFlightPerOpenDocument[document]++
		}
	}

	return urlMetadataLookupInFlight{
		amountOfLookedUpDocuments:     len(asksInFlightPerOpenDocument),
		asksInFlightPerOpenDocument:   asksInFlightPerOpenDocument,
		lookedUpDocumentsWithMetadata: distinctDocuments{},
	}
}

func (lookup *urlMetadataLookupInFlight) settle(outcome peerasks.URLMetadataAskOutcome) {
	if answeredAsk, answered := outcome.Answer.Get(); answered {
		lookup.answeredAsks = append(lookup.answeredAsks, answeredAsk)
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			lookup.settleWithMetadata(metadata.Hash)
		}
	}
	for _, document := range outcome.Ask.Documents {
		lookup.settleOneAskNaming(document)
	}
}

func (lookup *urlMetadataLookupInFlight) settleWithMetadata(document yacymodel.URLHash) {
	if _, open := lookup.asksInFlightPerOpenDocument[document]; !open {
		return
	}
	delete(lookup.asksInFlightPerOpenDocument, document)
	lookup.lookedUpDocumentsWithMetadata.add(document)
}

func (lookup *urlMetadataLookupInFlight) settleOneAskNaming(document yacymodel.URLHash) {
	if _, open := lookup.asksInFlightPerOpenDocument[document]; !open {
		return
	}
	lookup.asksInFlightPerOpenDocument[document]--
	if lookup.asksInFlightPerOpenDocument[document] == 0 {
		delete(lookup.asksInFlightPerOpenDocument, document)
	}
}

func (lookup *urlMetadataLookupInFlight) covered() bool {
	return len(lookup.lookedUpDocumentsWithMetadata) == lookup.amountOfLookedUpDocuments
}

func (lookup *urlMetadataLookupInFlight) settledShare() float64 {
	return float64(lookup.amountOfLookedUpDocuments-len(lookup.asksInFlightPerOpenDocument)) /
		float64(lookup.amountOfLookedUpDocuments)
}

func (lookup *urlMetadataLookupInFlight) endedBy(
	end URLMetadataLookupEnd,
) endedURLMetadataLookup {
	return endedURLMetadataLookup{answeredAsks: lookup.answeredAsks, end: end}
}

func (lookup *urlMetadataLookupInFlight) cutOff() endedURLMetadataLookup {
	return endedURLMetadataLookup{
		answeredAsks:                    lookup.answeredAsks,
		end:                             URLMetadataLookupEndedByCutoff,
		amountOfLookedUpDocumentsCutOff: len(lookup.asksInFlightPerOpenDocument),
	}
}
