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
	givenDocuments                yacymodel.URLHashes
	endReason                     EndReason
	amountOfDocumentsCutOff       int
}

func noAsksPutYet() *run {
	return &run{
		beganAt:                       time.Now(),
		lookedUpDocuments:             yacymodel.URLHashes{},
		asksInFlightPerOpenDocument:   map[yacymodel.URLHash]int{},
		lookedUpDocumentsWithMetadata: yacymodel.URLHashes{},
		givenDocuments:                yacymodel.URLHashes{},
	}
}

func (run *run) give(documents []yacymodel.URLHash) {
	run.givenDocuments.AddEach(documents)
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

func (run *run) endBy(endReason EndReason) {
	run.endReason = endReason
}

func (run *run) covered() bool {
	return len(run.lookedUpDocumentsWithMetadata) == len(run.lookedUpDocuments)
}

func (run *run) settle(outcome peerasks.URLMetadataAskOutcome) {
	if answeredAsk, answered := outcome.Answer.Get(); answered {
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

func (run *run) cutOff() {
	run.endReason = EndedByCutoff
	run.amountOfDocumentsCutOff = len(run.asksInFlightPerOpenDocument)
}

func (run *run) performed() Performed {
	return Performed{
		AmountOfLookedUpDocuments:             len(run.lookedUpDocuments),
		AmountOfLookedUpDocumentsWithMetadata: len(run.lookedUpDocumentsWithMetadata),
		AmountOfDocumentsNotAsked:             len(run.givenDocuments) - len(run.lookedUpDocuments),
		EndReason:                             run.endReason,
		AmountOfDocumentsCutOff:               run.amountOfDocumentsCutOff,
		AmountOfDocumentsPerAsk:               amountOfDocumentsPerAskIn(run.asks),
		TimeToFirstAsk:                        run.timeToFirstAsk,
	}
}

func amountOfDocumentsPerAskIn(asks []peerasks.URLMetadataAsk) []int {
	amountOfDocumentsPerAsk := make([]int, 0, len(asks))
	for _, ask := range asks {
		amountOfDocumentsPerAsk = append(amountOfDocumentsPerAsk, len(ask.Documents))
	}

	return amountOfDocumentsPerAsk
}
