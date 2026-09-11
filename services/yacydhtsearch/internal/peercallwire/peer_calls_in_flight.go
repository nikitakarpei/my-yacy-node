package peercallwire

type peerCallsInFlight chan struct{}

func (inFlight peerCallsInFlight) putOnePeerCall(peerCall func()) {
	inFlight <- struct{}{}
	defer func() { <-inFlight }()

	peerCall()
}
