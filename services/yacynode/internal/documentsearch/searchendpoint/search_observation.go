package searchendpoint

import (
	"context"
	"errors"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchresult"
)

type searchObservation struct {
	metrics      *searchmetrics.SearchMetrics
	nodePosition yacymodel.DHTPosition
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
	if len(result.DocumentMetadata) > 0 {
		o.metrics.ObserveSearchOutcome(searchmetrics.SearchServedWithResults)
	} else {
		o.metrics.ObserveSearchOutcome(searchmetrics.SearchServedNoResults)
	}
	for term, postingsHeld := range result.PostingsHeldPerTerm {
		fraction := yacymodel.PostingRingFractionToPosition(term, o.nodePosition, o.partitions)
		if postingsHeld > 0 {
			o.metrics.ObserveTermInIndex(fraction)
		} else {
			o.metrics.ObserveTermNotInIndex(fraction)
		}
	}
}
