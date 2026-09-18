package pagereading

import "time"

type PerformedPageReading struct {
	AmountOfPagesToRead              int
	AmountOfPagesRead                int
	AmountOfPagesUnreachable         int
	AmountOfPagesRefused             int
	AmountOfPagesUnreadable          int
	AmountOfPagesOfAnUnsupportedKind int
	AmountOfPagesOutOfBudget         int
	TimeSpent                        time.Duration
	TimeSpentFetching                time.Duration
	TimeSpentReading                 time.Duration
}

func performedPageReadingFrom(
	readPages []readPage,
	timeSpent time.Duration,
) PerformedPageReading {
	performed := PerformedPageReading{
		AmountOfPagesToRead: len(readPages),
		TimeSpent:           timeSpent,
	}
	for _, readPage := range readPages {
		performed.TimeSpentFetching += readPage.timeSpentFetching
		performed.TimeSpentReading += readPage.timeSpentReading
		switch readPage.outcome {
		case pageWasRead:
			performed.AmountOfPagesRead++
		case pageWasUnreachable:
			performed.AmountOfPagesUnreachable++
		case pageWasRefused:
			performed.AmountOfPagesRefused++
		case pageWasUnreadable:
			performed.AmountOfPagesUnreadable++
		case pageWasOfAnUnsupportedKind:
			performed.AmountOfPagesOfAnUnsupportedKind++
		case pageWasOutOfBudget:
			performed.AmountOfPagesOutOfBudget++
		}
	}

	return performed
}
