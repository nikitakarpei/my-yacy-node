package pagereading

import (
	"math"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PageReadCutoff struct {
	PercentOfPages int
	Grace          time.Duration
}

func (cutoff PageReadCutoff) pageReadResultsFrom(
	clock Clock,
	budgetEnded <-chan struct{},
	settlingResults <-chan pageReadResult,
	pagesWanted []PageToRead,
) []pageReadResult {
	pageReadResults := make([]pageReadResult, 0, len(pagesWanted))
	settledDocuments := make(map[yacymodel.URLHash]struct{}, len(pagesWanted))
	amountOfPagesForTheGrace := int(
		math.Ceil(float64(cutoff.PercentOfPages) / 100 * float64(len(pagesWanted))),
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
				stopGrace = clock.After(cutoff.Grace, func() { close(graceEnded) })
			}
		case <-graceEnded:
			return withUnsettledPagesAs(
				pageWasCutOff,
				pageReadResults,
				settledDocuments,
				pagesWanted,
			)
		case <-budgetEnded:
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
