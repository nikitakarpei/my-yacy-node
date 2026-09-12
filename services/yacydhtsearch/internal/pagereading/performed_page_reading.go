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
		switch readPage.outcome {
		case pageRead:
			performed.AmountOfPagesRead++
		case pageUnreachable:
			performed.AmountOfPagesUnreachable++
		case pageRefused:
			performed.AmountOfPagesRefused++
		case pageUnreadable:
			performed.AmountOfPagesUnreadable++
		case pageOfAnUnsupportedKind:
			performed.AmountOfPagesOfAnUnsupportedKind++
		case pageOutOfBudget:
			performed.AmountOfPagesOutOfBudget++
		}
	}

	return performed
}
