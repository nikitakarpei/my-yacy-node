package pagereading

import (
	"context"
	"math"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PageReadCutoff struct {
	PercentOfPages int
	Grace          time.Duration
}

type pageReadWaiting struct {
	budget time.Duration
	cutoff PageReadCutoff
	clock  Clock
}

func (waiting pageReadWaiting) pageReadResultsFrom(
	ctx context.Context,
	settlingResults <-chan pageReadResult,
	pagesWanted []PageToRead,
) []pageReadResult {
	budgetedCtx, stopBudget := context.WithTimeout(ctx, waiting.budget)
	defer stopBudget()
	pageReadResults := make([]pageReadResult, 0, len(pagesWanted))
	settledDocuments := make(map[yacymodel.URLHash]struct{}, len(pagesWanted))
	amountOfPagesForTheGrace := int(
		math.Ceil(float64(waiting.cutoff.PercentOfPages) / 100 * float64(len(pagesWanted))),
	)
	var graceEnded chan struct{}
	stopGrace := func() {}
	defer func() { stopGrace() }()
	for len(pageReadResults) < len(pagesWanted) {
		select {
		case result := <-settlingResults:
			pageReadResults = append(pageReadResults, result)
			settledDocuments[result.document] = struct{}{}
			if len(pageReadResults) == amountOfPagesForTheGrace {
				graceEnded = make(chan struct{})
				stopGrace = waiting.clock.After(waiting.cutoff.Grace, func() { close(graceEnded) })
			}
		case <-graceEnded:
			return withUnsettledPagesAs(
				pageWasCutOff,
				pageReadResults,
				settledDocuments,
				pagesWanted,
			)
		case <-budgetedCtx.Done():
			return withUnsettledPagesAs(
				pageWasOutOfBudget,
				pageReadResults,
				settledDocuments,
				pagesWanted,
			)
		}
	}

	return pageReadResults
}

func withUnsettledPagesAs(
	outcome readOutcome,
	pageReadResults []pageReadResult,
	settledDocuments map[yacymodel.URLHash]struct{},
	pagesWanted []PageToRead,
) []pageReadResult {
	for _, pageWanted := range pagesWanted {
		if _, settled := settledDocuments[pageWanted.Document]; !settled {
			pageReadResults = append(pageReadResults, pageReadResult{
				document: pageWanted.Document,
				outcome:  outcome,
			})
		}
	}

	return pageReadResults
}
