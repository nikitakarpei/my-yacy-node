package peercallwire

import "sync"

type peerCallsInFlight chan struct{}

func (inFlight peerCallsInFlight) putEveryPeerCallInTheOrderGiven(
	amountOfPeerCalls int,
	putOnePeerCall func(index int),
) {
	var peerCalls sync.WaitGroup
	for index := range amountOfPeerCalls {
		inFlight <- struct{}{}
		peerCalls.Add(1)
		go func() {
			defer peerCalls.Done()
			defer func() { <-inFlight }()

			putOnePeerCall(index)
		}()
	}
	peerCalls.Wait()
}
