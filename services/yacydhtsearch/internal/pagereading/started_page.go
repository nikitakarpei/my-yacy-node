package pagereading

import "context"

type startedPage struct {
	pageToRead     PageToRead
	readContext    context.Context //nolint:containedctx // the read of one page ends at its budget or when the page is abandoned
	stopReading    context.CancelFunc
	settled        chan struct{}
	pageReadResult pageReadResult
}

func startedPageOf(ctx context.Context, pageToRead PageToRead) *startedPage {
	readContext, stopReading := context.WithCancel(ctx)

	return &startedPage{
		pageToRead:  pageToRead,
		readContext: readContext,
		stopReading: stopReading,
		settled:     make(chan struct{}),
	}
}

func (page *startedPage) settleWith(result pageReadResult) {
	page.pageReadResult = result
	close(page.settled)
}

func (page *startedPage) pageReadResultOnceSettled() pageReadResult {
	<-page.settled

	return page.pageReadResult
}

func (page *startedPage) isSettled() bool {
	select {
	case <-page.settled:
		return true
	default:
		return false
	}
}
