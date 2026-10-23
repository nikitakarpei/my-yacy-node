package consolepages

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type visitOutcome int

const (
	noVisitOutcome visitOutcome = iota
	publishedOutcome
	unchangedOutcome
	unusableContentOutcome
	overPageLimitOutcome
	refusedBySiteOutcome
	deadLinksOutcome
	serverErrorsOutcome
)

var visitOutcomeOfResult = map[yacycrawlcontract.PageVisitResult]visitOutcome{
	yacycrawlcontract.ResultPublished:             publishedOutcome,
	yacycrawlcontract.ResultRedirected:            noVisitOutcome,
	yacycrawlcontract.ResultNotModified:           unchangedOutcome,
	yacycrawlcontract.ResultAccessRefused:         refusedBySiteOutcome,
	yacycrawlcontract.ResultFetchRejected:         refusedBySiteOutcome,
	yacycrawlcontract.ResultNotFound:              deadLinksOutcome,
	yacycrawlcontract.ResultRedirectTargetInvalid: deadLinksOutcome,
	yacycrawlcontract.ResultOversized:             unusableContentOutcome,
	yacycrawlcontract.ResultUnsupportedMediaType:  unusableContentOutcome,
	yacycrawlcontract.ResultUnreadableHTML:        unusableContentOutcome,
}

var visitOutcomeOfDropReason = map[yacycrawlcontract.PageVisitDropReason]visitOutcome{
	yacycrawlcontract.DropReasonNotDue:             unchangedOutcome,
	yacycrawlcontract.DropReasonHostPagesExhausted: overPageLimitOutcome,
	yacycrawlcontract.DropReasonDeferralsExhausted: refusedBySiteOutcome,
	yacycrawlcontract.DropReasonRetriesExhausted:   serverErrorsOutcome,
	yacycrawlcontract.DropReasonCrawlCancelled:     noVisitOutcome,
}

type visitOutcomes struct {
	Published       int64
	Unchanged       int64
	UnusableContent int64
	OverPageLimit   int64
	RefusedBySite   int64
	DeadLinks       int64
	ServerErrors    int64
}

func visitOutcomesFrom(
	completedByResult map[yacycrawlcontract.PageVisitResult]int64,
	droppedByReason map[yacycrawlcontract.PageVisitDropReason]int64,
) visitOutcomes {
	var outcomes visitOutcomes
	for result, amount := range completedByResult {
		outcomes.add(visitOutcomeOfResult[result], amount)
	}
	for reason, amount := range droppedByReason {
		outcomes.add(visitOutcomeOfDropReason[reason], amount)
	}
	return outcomes
}

func (outcomes *visitOutcomes) add(outcome visitOutcome, amount int64) {
	switch outcome {
	case publishedOutcome:
		outcomes.Published += amount
	case unchangedOutcome:
		outcomes.Unchanged += amount
	case unusableContentOutcome:
		outcomes.UnusableContent += amount
	case overPageLimitOutcome:
		outcomes.OverPageLimit += amount
	case refusedBySiteOutcome:
		outcomes.RefusedBySite += amount
	case deadLinksOutcome:
		outcomes.DeadLinks += amount
	case serverErrorsOutcome:
		outcomes.ServerErrors += amount
	case noVisitOutcome:
	}
}

func (outcomes visitOutcomes) NotFetched() int64 {
	return outcomes.UnusableContent + outcomes.OverPageLimit + outcomes.RefusedBySite +
		outcomes.DeadLinks + outcomes.ServerErrors
}
