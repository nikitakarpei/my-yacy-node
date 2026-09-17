package peercallwire

import (
	"context"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type peerCallsInFlight struct {
	slots    chan struct{}
	observer PeerCallObserver
}

func newPeerCallsInFlight(amountOfSlots int, observer PeerCallObserver) peerCallsInFlight {
	return peerCallsInFlight{
		slots:    make(chan struct{}, amountOfSlots),
		observer: observer,
	}
}

func (inFlight peerCallsInFlight) putEveryPeerCallInTheOrderGiven(
	ctx context.Context,
	amountOfPeerCalls int,
	askedPeerAt func(index int) (address string, askedFor peerasks.AskedFor),
	putOnePeerCall func(index int),
) {
	var peerCalls sync.WaitGroup
	for index := range amountOfPeerCalls {
		address, askedFor := askedPeerAt(index)
		inFlight.takeASlot(ctx, address, askedFor)
		peerCalls.Add(1)
		go func() {
			defer peerCalls.Done()
			defer func() { <-inFlight.slots }()

			putOnePeerCall(index)
		}()
	}
	peerCalls.Wait()
}

func (inFlight peerCallsInFlight) takeASlot(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
) {
	inFlight.observer.PeerCallWaitsForASlot(ctx, address, askedFor)
	waitStartedAt := time.Now()
	inFlight.slots <- struct{}{}
	inFlight.observer.PeerCallTookASlot(ctx, address, askedFor, time.Since(waitStartedAt))
}
