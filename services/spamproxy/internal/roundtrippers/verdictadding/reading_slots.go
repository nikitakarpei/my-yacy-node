package verdictadding

import "context"

type readingSlots chan struct{}

func (s readingSlots) reserve(readingCtx context.Context) bool {
	select {
	case s <- struct{}{}:
		return true
	default:
	}
	select {
	case s <- struct{}{}:
		return true
	case <-readingCtx.Done():
		return false
	}
}

func (s readingSlots) release() {
	<-s
}
