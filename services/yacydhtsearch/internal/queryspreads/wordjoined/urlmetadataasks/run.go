package urlmetadataasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type run struct {
	asks                        []peerasks.URLMetadataAsk
	amountOfAskedDocuments      int
	asksInFlightPerOpenDocument map[yacymodel.URLHash]int
	askedDocumentsWithMetadata  yacymodel.URLHashes
	answeredAsks                []peerasks.AnsweredURLMetadataAsk
}

func runOf(asks []peerasks.URLMetadataAsk) *run {
	asksInFlightPerOpenDocument := map[yacymodel.URLHash]int{}
	for _, ask := range asks {
		for _, document := range ask.Documents {
			asksInFlightPerOpenDocument[document]++
		}
	}

	return &run{
		asks:                        asks,
		amountOfAskedDocuments:      len(asksInFlightPerOpenDocument),
		asksInFlightPerOpenDocument: asksInFlightPerOpenDocument,
		askedDocumentsWithMetadata:  yacymodel.URLHashes{},
	}
}

func (asks *run) settleUntilEnded(
	outcomesAsTheySettle <-chan peerasks.URLMetadataAskOutcome,
	cutoff Cutoff,
	clock Clock,
) Answers {
	graceEnded := make(chan struct{})
	stopGrace := func() {}
	defer func() { stopGrace() }()
	graceStarted := false
	for {
		select {
		case outcome, open := <-outcomesAsTheySettle:
			if !open {
				return asks.endedBy(EndedByEveryAskSettled)
			}
			asks.settle(outcome)
			if asks.covered() {
				return asks.endedBy(EndedByCoverage)
			}
			if !graceStarted && cutoff.reachedBy(asks.settledShare()) {
				graceStarted = true
				stopGrace = clock.After(cutoff.Grace, func() { close(graceEnded) })
			}
		case <-graceEnded:
			return asks.cutOff()
		}
	}
}

func (asks *run) endedBy(endReason EndReason) Answers {
	return Answers{asks: asks.asks, answeredAsks: asks.answeredAsks, endReason: endReason}
}

func (asks *run) settle(outcome peerasks.URLMetadataAskOutcome) {
	if answeredAsk, answered := outcome.Answer.Get(); answered {
		asks.answeredAsks = append(asks.answeredAsks, answeredAsk)
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			asks.settleWithMetadata(metadata.Hash)
		}
	}
	for _, document := range outcome.Ask.Documents {
		asks.settleOneAskNaming(document)
	}
}

func (asks *run) settleWithMetadata(document yacymodel.URLHash) {
	if _, open := asks.asksInFlightPerOpenDocument[document]; !open {
		return
	}
	delete(asks.asksInFlightPerOpenDocument, document)
	asks.askedDocumentsWithMetadata.Add(document)
}

func (asks *run) settleOneAskNaming(document yacymodel.URLHash) {
	if _, open := asks.asksInFlightPerOpenDocument[document]; !open {
		return
	}
	asks.asksInFlightPerOpenDocument[document]--
	if asks.asksInFlightPerOpenDocument[document] == 0 {
		delete(asks.asksInFlightPerOpenDocument, document)
	}
}

func (asks *run) covered() bool {
	return len(asks.askedDocumentsWithMetadata) == asks.amountOfAskedDocuments
}

func (asks *run) settledShare() float64 {
	return float64(asks.amountOfAskedDocuments-len(asks.asksInFlightPerOpenDocument)) /
		float64(asks.amountOfAskedDocuments)
}

func (asks *run) cutOff() Answers {
	return Answers{
		asks:                    asks.asks,
		answeredAsks:            asks.answeredAsks,
		endReason:               EndedByCutoff,
		amountOfDocumentsCutOff: len(asks.asksInFlightPerOpenDocument),
	}
}
