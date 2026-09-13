package searchendpoint

import (
	"context"
	"errors"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/documentmatch"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchresult"
)

type searchObservation struct {
	metrics      *searchmetrics.SearchMetrics
	nodePosition yacymodel.DHTRingPosition
	partitions   yacymodel.DHTRingPartitions
}

func (o searchObservation) observeNetworkMismatch() {
	o.metrics.ObserveSearchOutcome(searchmetrics.SearchNetworkMismatch)
}

func (o searchObservation) observeInvalidCriteria() {
	o.metrics.ObserveSearchOutcome(searchmetrics.SearchInvalidCriteria)
}

func (o searchObservation) observeIgnoredOptions(options []string) {
	for _, option := range options {
		o.metrics.ObserveUnsupportedOptionRequested(option)
	}
}

func (o searchObservation) observeSearchFailure(err error) {
	o.metrics.ObserveSearchOutcome(outcomeOfSearchFailure(err))
}

func outcomeOfSearchFailure(err error) searchmetrics.SearchOutcome {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return searchmetrics.SearchDeadlineExceeded
	case errors.Is(err, searchresult.ErrDocumentDirectory):
		return searchmetrics.SearchMetadataFailure
	default:
		return searchmetrics.SearchIndexFailure
	}
}

func (o searchObservation) observeServed(result searchresult.Result) {
	if len(result.MatchedDocuments) > 0 {
		o.metrics.ObserveSearchOutcome(searchmetrics.SearchServedWithResults)
	} else {
		o.metrics.ObserveSearchOutcome(searchmetrics.SearchServedNoResults)
	}
	o.metrics.ObserveIndexReadStopReason(indexReadStopReasonOf(result.IndexReadStopReason))
	for term, amountOfPostings := range result.AmountOfPostingsPerTerm {
		nearness := o.nodePosition.DistanceFromPostingsOfWord(term, o.partitions)
		if amountOfPostings > 0 {
			o.metrics.ObserveTermInIndex(nearness.FractionOfDHTRing())
		} else {
			o.metrics.ObserveTermNotInIndex(nearness.FractionOfDHTRing())
		}
	}
}

func indexReadStopReasonOf(
	reason documentmatch.IndexReadStopReason,
) searchmetrics.IndexReadStopReason {
	switch reason {
	case documentmatch.IndexReadStoppedAtRelevanceBound:
		return searchmetrics.IndexReadStoppedAtRelevanceBound
	case documentmatch.IndexReadStoppedAtDeadline:
		return searchmetrics.IndexReadStoppedAtDeadline
	default:
		return searchmetrics.IndexReadStoppedAtEndOfTerm
	}
}
