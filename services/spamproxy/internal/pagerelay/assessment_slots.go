package pagerelay

type assessmentSlots chan struct{}

func (s assessmentSlots) tryReserve() bool {
	select {
	case s <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s assessmentSlots) release() {
	<-s
}
