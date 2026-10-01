package pagereading

type startedPage struct {
	settled        chan struct{}
	pageReadResult pageReadResult
}

func (page *startedPage) settleWith(result pageReadResult) {
	page.pageReadResult = result
	close(page.settled)
}

func (page *startedPage) pageReadResultOnceSettled() pageReadResult {
	<-page.settled

	return page.pageReadResult
}
