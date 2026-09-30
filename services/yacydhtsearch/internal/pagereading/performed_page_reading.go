package pagereading

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
)

type PerformedPageReading struct {
	AmountOfPagesToRead              int
	AmountOfPagesRead                int
	AmountOfPagesUnreachable         int
	AmountOfPagesRefused             int
	AmountOfPagesGone                int
	AmountOfPagesRefusingIndexing    int
	AmountOfPagesUnreadable          int
	AmountOfPagesOfAnUnsupportedKind int
	AmountOfPagesOutOfBudget         int
	AmountOfPagesCutOff              int
	AmountOfPagesReadPerSpamVerdict  map[spamassessment.Verdict]int
	TimeSpent                        time.Duration
	TimeSpentFetching                time.Duration
	TimeSpentReading                 time.Duration
}

func performedPageReadingFrom(
	pageReadResults []pageReadResult,
	timeSpent time.Duration,
) PerformedPageReading {
	performed := PerformedPageReading{
		AmountOfPagesToRead:             len(pageReadResults),
		AmountOfPagesReadPerSpamVerdict: map[spamassessment.Verdict]int{},
		TimeSpent:                       timeSpent,
	}
	for _, pageReadResult := range pageReadResults {
		performed.TimeSpentFetching += pageReadResult.timeSpentFetching
		performed.TimeSpentReading += pageReadResult.timeSpentReading
		switch pageReadResult.outcome {
		case pageWasRead:
			performed.AmountOfPagesRead++
			performed.AmountOfPagesReadPerSpamVerdict[pageReadResult.spamVerdict]++
		case pageWasUnreachable:
			performed.AmountOfPagesUnreachable++
		case pageWasRefused:
			performed.AmountOfPagesRefused++
		case pageWasGone:
			performed.AmountOfPagesGone++
		case pageRefusesIndexing:
			performed.AmountOfPagesRefusingIndexing++
		case pageWasUnreadable:
			performed.AmountOfPagesUnreadable++
		case pageWasOfAnUnsupportedKind:
			performed.AmountOfPagesOfAnUnsupportedKind++
		case pageWasOutOfBudget:
			performed.AmountOfPagesOutOfBudget++
		case pageWasCutOff:
			performed.AmountOfPagesCutOff++
		}
	}

	return performed
}
