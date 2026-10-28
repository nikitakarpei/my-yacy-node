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
		beganAt:                     time.Now(),
		asksInFlightPerOpenDocument: map[yacymodel.URLHash]int{},
	}
}

func (run *run) give(documents []yacymodel.URLHash) {
	run.givenDocuments = run.givenDocuments.UnionWith(yacymodel.URLHashesOf(documents...))
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
	var lookedUpDocuments yacymodel.URLHashesBuilder
	for _, ask := range asks {
		for _, document := range ask.Documents {
			lookedUpDocuments.Add(document)
			run.asksInFlightPerOpenDocument[document]++
		}
	}
	run.lookedUpDocuments = run.lookedUpDocuments.UnionWith(lookedUpDocuments.URLHashes())
}

func (run *run) endBy(endReason EndReason) {
	run.endReason = endReason
}

func (run *run) covered() bool {
	return run.lookedUpDocumentsWithMetadata.Amount() == run.lookedUpDocuments.Amount()
}

func (run *run) settle(outcome peerasks.URLMetadataAskOutcome) {
	if answeredAsk, answered := outcome.Answer.Get(); answered {
		run.settleWithMetadata(answeredAsk.MetadataOfEachDocument)
	}
	for _, document := range outcome.Ask.Documents {
		run.settleOneAskNaming(document)
	}
}

func (run *run) settleWithMetadata(metadataOfEachDocument []yacymodel.URLMetadata) {
	var documentsWithMetadata yacymodel.URLHashesBuilder
	for _, metadata := range metadataOfEachDocument {
		if _, open := run.asksInFlightPerOpenDocument[metadata.Hash]; !open {
			continue
		}
		delete(run.asksInFlightPerOpenDocument, metadata.Hash)
		documentsWithMetadata.Add(metadata.Hash)
	}
	run.lookedUpDocumentsWithMetadata = run.lookedUpDocumentsWithMetadata.UnionWith(
		documentsWithMetadata.URLHashes(),
	)
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
	return float64(run.lookedUpDocuments.Amount()-len(run.asksInFlightPerOpenDocument)) /
		float64(run.lookedUpDocuments.Amount())
}

func (run *run) cutOff() {
	run.endReason = EndedByCutoff
	run.amountOfDocumentsCutOff = len(run.asksInFlightPerOpenDocument)
}

func (run *run) performed() Performed {
	return Performed{
		AmountOfLookedUpDocuments:             run.lookedUpDocuments.Amount(),
		AmountOfLookedUpDocumentsWithMetadata: run.lookedUpDocumentsWithMetadata.Amount(),
		AmountOfDocumentsNotAsked:             run.givenDocuments.Amount() - run.lookedUpDocuments.Amount(),
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
