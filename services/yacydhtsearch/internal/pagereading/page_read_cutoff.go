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
	settledPages <-chan settledPage,
	pagesToRead []PageToRead,
) []pageReadResult {
	pageReadResults := make([]pageReadResult, len(pagesToRead))
	settled := make([]bool, len(pagesToRead))
	amountOfPagesForTheGrace := int(
		math.Ceil(float64(cutoff.PercentOfPages) / 100 * float64(len(pagesToRead))),
	)
	var graceEnded <-chan time.Time
	for amountOfSettledPages := 0; amountOfSettledPages < len(pagesToRead); {
		select {
		case page := <-settledPages:
			pageReadResults[page.place] = page.pageReadResult
			settled[page.place] = true
			amountOfSettledPages++
			if amountOfSettledPages == amountOfPagesForTheGrace {
				graceEnded = time.After(cutoff.Grace)
			}
		case <-graceEnded:
			return withPagesCutOff(pageReadResults, settled, pagesToRead)
		}
	}

	return pageReadResults
}

func withPagesCutOff(
	pageReadResults []pageReadResult,
	settled []bool,
	pagesToRead []PageToRead,
) []pageReadResult {
	for place, pageToRead := range pagesToRead {
		if !settled[place] {
			pageReadResults[place] = pageReadResult{
				document: pageToRead.Document,
				outcome:  pageWasCutOff,
			}
		}
	}

	return pageReadResults
}
