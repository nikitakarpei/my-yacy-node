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

func (run *run) waitUntilEnded(
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
				return run.endedBy(EndedByEveryAskSettled)
			}
			run.settle(outcome)
			if run.covered() {
				return run.endedBy(EndedByCoverage)
			}
			if !graceStarted && cutoff.reachedBy(run.settledShare()) {
				graceStarted = true
				stopGrace = clock.After(cutoff.Grace, func() { close(graceEnded) })
			}
		case <-graceEnded:
			return run.cutOff()
		}
	}
}

func (run *run) endedBy(endReason EndReason) Answers {
	return Answers{asks: run.asks, answeredAsks: run.answeredAsks, endReason: endReason}
}

func (run *run) settle(outcome peerasks.URLMetadataAskOutcome) {
	if answeredAsk, answered := outcome.Answer.Get(); answered {
		run.answeredAsks = append(run.answeredAsks, answeredAsk)
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			run.settleWithMetadata(metadata.Hash)
		}
	}
	for _, document := range outcome.Ask.Documents {
		run.settleOneAskNaming(document)
	}
}

func (run *run) settleWithMetadata(document yacymodel.URLHash) {
	if _, open := run.asksInFlightPerOpenDocument[document]; !open {
		return
	}
	delete(run.asksInFlightPerOpenDocument, document)
	run.askedDocumentsWithMetadata.Add(document)
}

func (run *run) settleOneAskNaming(document yacymodel.URLHash) {
	if _, open := run.asksInFlightPerOpenDocument[document]; !open {
		return
	}
	run.asksInFlightPerOpenDocument[document]--
	if run.asksInFlightPerOpenDocument[document] == 0 {
		delete(run.asksInFlightPerOpenDocument, document)
	}
}

func (run *run) covered() bool {
	return len(run.askedDocumentsWithMetadata) == run.amountOfAskedDocuments
}

func (run *run) settledShare() float64 {
	return float64(run.amountOfAskedDocuments-len(run.asksInFlightPerOpenDocument)) /
		float64(run.amountOfAskedDocuments)
}

func (run *run) cutOff() Answers {
	return Answers{
		asks:                    run.asks,
		answeredAsks:            run.answeredAsks,
		endReason:               EndedByCutoff,
		amountOfDocumentsCutOff: len(run.asksInFlightPerOpenDocument),
	}
}
