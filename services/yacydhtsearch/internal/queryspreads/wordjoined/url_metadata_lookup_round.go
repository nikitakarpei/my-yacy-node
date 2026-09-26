package wordjoined

import (
	"maps"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type urlMetadataLookupRound struct {
	documentsWithoutMetadata distinctDocuments
	asks                     []peerasks.URLMetadataAsk
	endedURLMetadataLookup
}

type endedURLMetadataLookup struct {
	answeredAsks                        []peerasks.AnsweredURLMetadataAsk
	endReason                           URLMetadataLookupEndReason
	amountOfDocumentsCutOffDuringLookup int
}

type URLMetadataLookupEndReason string

const (
	URLMetadataLookupEndedByCoverage        URLMetadataLookupEndReason = "coverage"
	URLMetadataLookupEndedByEveryAskSettled URLMetadataLookupEndReason = "every ask settled"
	URLMetadataLookupEndedByCutoff          URLMetadataLookupEndReason = "cut off"
)

func documentsWithoutMetadataAmong(
	joinedDocuments distinctDocuments,
	answers []wordpartitionasks.ReplicaAnswer,
) distinctDocuments {
	documentsWithoutMetadata := maps.Clone(joinedDocuments)
	for _, answer := range answers {
		for _, listedDocument := range answer.ListedDocuments {
			if !listedDocument.Metadata.Present() {
				continue
			}
			delete(documentsWithoutMetadata, listedDocument.Hash)
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

func (lookup *urlMetadataLookupInFlight) settleUntilEnded(
	outcomesAsTheySettle <-chan peerasks.URLMetadataAskOutcome,
	cutoff URLMetadataLookupCutoff,
) endedURLMetadataLookup {
	var graceEnded <-chan time.Time
	for {
		select {
		case outcome, open := <-outcomesAsTheySettle:
			if !open {
				return lookup.endedBy(URLMetadataLookupEndedByEveryAskSettled)
			}
			lookup.settle(outcome)
			if lookup.covered() {
				return lookup.endedBy(URLMetadataLookupEndedByCoverage)
			}
			if graceEnded == nil && cutoff.reachedBy(lookup.settledShare()) {
				grace := time.NewTimer(cutoff.Grace)
				defer grace.Stop()
				graceEnded = grace.C
			}
		case <-graceEnded:
			return lookup.cutOff()
		}
	}
}

func (lookup *urlMetadataLookupInFlight) endedBy(
	endReason URLMetadataLookupEndReason,
) endedURLMetadataLookup {
	return endedURLMetadataLookup{answeredAsks: lookup.answeredAsks, endReason: endReason}
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

func (lookup *urlMetadataLookupInFlight) cutOff() endedURLMetadataLookup {
	return endedURLMetadataLookup{
		answeredAsks:                        lookup.answeredAsks,
		endReason:                           URLMetadataLookupEndedByCutoff,
		amountOfDocumentsCutOffDuringLookup: len(lookup.asksInFlightPerOpenDocument),
	}
}
