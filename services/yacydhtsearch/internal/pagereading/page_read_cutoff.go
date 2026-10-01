package pagereading

import (
	"math"
	"time"
)

type PageReadCutoff struct {
	PercentOfPages int
	Grace          time.Duration
}

func (cutoff PageReadCutoff) pageReadResultsFrom(
	clock Clock,
	settledPages <-chan settledPage,
	pagesWanted []PageToRead,
) []pageReadResult {
	pageReadResults := make([]pageReadResult, len(pagesWanted))
	settled := make([]bool, len(pagesWanted))
	amountOfPagesForTheGrace := int(
		math.Ceil(float64(cutoff.PercentOfPages) / 100 * float64(len(pagesWanted))),
	)
	var graceEnded chan struct{}
	stopGrace := func() {}
	defer func() { stopGrace() }()
	for amountOfSettledPages := 0; amountOfSettledPages < len(pagesWanted); {
		select {
		case page := <-settledPages:
			pageReadResults[page.place] = page.pageReadResult
			settled[page.place] = true
			amountOfSettledPages++
			if amountOfSettledPages == amountOfPagesForTheGrace {
				graceEnded = make(chan struct{})
				stopGrace = clock.After(cutoff.Grace, func() { close(graceEnded) })
			}
		case <-graceEnded:
			return withPagesCutOff(pageReadResults, settled, pagesWanted)
		}
	}

	return pageReadResults
}

func withPagesCutOff(
	pageReadResults []pageReadResult,
	settled []bool,
	pagesWanted []PageToRead,
) []pageReadResult {
	for place, pageWanted := range pagesWanted {
		if !settled[place] {
			pageReadResults[place] = pageReadResult{
				document: pageWanted.Document,
				outcome:  pageWasCutOff,
			}
		}
	}

	return pageReadResults
}
