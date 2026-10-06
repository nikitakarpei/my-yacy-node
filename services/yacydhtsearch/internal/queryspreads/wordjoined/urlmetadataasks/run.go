package urlmetadataasks

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type run struct {
	beganAt                       time.Time
	timeToFirstAsk                yacymodel.Optional[time.Duration]
	asks                          []peerasks.URLMetadataAsk
	lookedUpDocuments             yacymodel.URLHashes
	asksInFlightPerOpenDocument   map[yacymodel.URLHash]int
	lookedUpDocumentsWithMetadata yacymodel.URLHashes
	answeredAsks                  []peerasks.AnsweredURLMetadataAsk
}

func noAsksPutYet() *run {
	return &run{
		beganAt:                       time.Now(),
		lookedUpDocuments:             yacymodel.URLHashes{},
		asksInFlightPerOpenDocument:   map[yacymodel.URLHash]int{},
		lookedUpDocumentsWithMetadata: yacymodel.URLHashes{},
	}
}

func (run *run) add(asks []peerasks.URLMetadataAsk) {
	run.timeTheFirstAsk()
	run.keep(asks)
	run.openTheDocumentsOf(asks)
}

func (run *run) timeTheFirstAsk() {
	if run.timeToFirstAsk.Present() {
		return
	}
	run.timeToFirstAsk = yacymodel.Some(time.Since(run.beganAt))
}

func (run *run) keep(asks []peerasks.URLMetadataAsk) {
	run.asks = append(run.asks, asks...)
}

func (run *run) openTheDocumentsOf(asks []peerasks.URLMetadataAsk) {
	for _, ask := range asks {
		for _, document := range ask.Documents {
			run.lookedUpDocuments.Add(document)
			run.asksInFlightPerOpenDocument[document]++
		}
	}
}

func (run *run) endedBy(endReason EndReason) Answers {
	return Answers{
		asks:           run.asks,
		answeredAsks:   run.answeredAsks,
		endReason:      endReason,
		timeToFirstAsk: run.timeToFirstAsk,
	}
}

func (run *run) covered() bool {
	return len(run.lookedUpDocumentsWithMetadata) == len(run.lookedUpDocuments)
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
	run.lookedUpDocumentsWithMetadata.Add(document)
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

func (run *run) settledShare() float64 {
	return float64(len(run.lookedUpDocuments)-len(run.asksInFlightPerOpenDocument)) /
		float64(len(run.lookedUpDocuments))
}

func (run *run) cutOff() Answers {
	return Answers{
		asks:                    run.asks,
		answeredAsks:            run.answeredAsks,
		endReason:               EndedByCutoff,
		amountOfDocumentsCutOff: len(run.asksInFlightPerOpenDocument),
		timeToFirstAsk:          run.timeToFirstAsk,
	}
}
