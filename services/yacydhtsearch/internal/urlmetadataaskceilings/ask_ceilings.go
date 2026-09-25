// Package urlmetadataaskceilings holds, for each peer address, the most
// documents one URL metadata ask may name. It sizes each ask to the documents
// the peer answers within a target time, from a moving average of what its
// past calls showed, between a floor and a ceiling.
package urlmetadataaskceilings

import (
	"context"
	"math"
	"sync"
	"time"
)

const (
	weightOfTheNewSample = 0.5
	shortestSpentTime    = time.Millisecond
)

type AskCeilings struct {
	mutex                           sync.Mutex
	mostDocuments                   int
	leastDocuments                  int
	targetTime                      time.Duration
	documentsPerSecondOfEachAddress map[string]float64
	ceilingHandedOutOfEachAddress   map[string]int
	lastAskOfEachAddress            map[string]lastAsk
	observer                        AskCeilingObserver
}

type lastAsk struct {
	documentsAsked      int
	limitedByTheCeiling bool
}

func New(
	mostDocuments, leastDocuments int,
	targetTime time.Duration,
	observer AskCeilingObserver,
) *AskCeilings {
	return &AskCeilings{
		mostDocuments:                   mostDocuments,
		leastDocuments:                  min(leastDocuments, mostDocuments),
		targetTime:                      targetTime,
		documentsPerSecondOfEachAddress: map[string]float64{},
		ceilingHandedOutOfEachAddress:   map[string]int{},
		lastAskOfEachAddress:            map[string]lastAsk{},
		observer:                        observer,
	}
}

func (ceilings *AskCeilings) CeilingOf(ctx context.Context, address string) (int, bool) {
	ceilings.mutex.Lock()
	defer ceilings.mutex.Unlock()

	ceiling := ceilings.ceilingOfAddress(address)
	ceilings.ceilingHandedOutOfEachAddress[address] = ceiling
	ceilings.observer.AskCeilingHandedOut(ctx, address, ceiling)

	return ceiling, ceiling < ceilings.mostDocuments
}

func (ceilings *AskCeilings) ceilingOfAddress(address string) int {
	documentsPerSecond, sampled := ceilings.documentsPerSecondOfEachAddress[address]
	if !sampled {
		return ceilings.mostDocuments
	}

	return ceilings.ceilingAt(documentsPerSecond)
}

func (ceilings *AskCeilings) ceilingAt(documentsPerSecond float64) int {
	documentsWithinTheTargetTime := math.Round(documentsPerSecond * ceilings.targetTime.Seconds())

	return int(max(
		float64(ceilings.leastDocuments),
		min(float64(ceilings.mostDocuments), documentsWithinTheTargetTime),
	))
}

func (ceilings *AskCeilings) Asked(address string, amountOfDocuments int) {
	ceilings.mutex.Lock()
	defer ceilings.mutex.Unlock()

	ceilingHandedOut, handedOut := ceilings.ceilingHandedOutOfEachAddress[address]
	if !handedOut {
		ceilingHandedOut = ceilings.ceilingOfAddress(address)
	}
	delete(ceilings.ceilingHandedOutOfEachAddress, address)
	ceilings.lastAskOfEachAddress[address] = lastAsk{
		documentsAsked:      amountOfDocuments,
		limitedByTheCeiling: amountOfDocuments >= ceilingHandedOut,
	}
}

func (ceilings *AskCeilings) AskAnswered(ctx context.Context, address string, spent time.Duration) {
	ceilings.mutex.Lock()
	defer ceilings.mutex.Unlock()

	ask, asked := ceilings.lastAskTakenFrom(address)
	if !asked || !ask.limitedByTheCeiling {
		return
	}
	ceilings.addSample(ctx, address, ceilings.documentsPerSecondOf(ask.documentsAsked, spent))
}

func (ceilings *AskCeilings) lastAskTakenFrom(address string) (lastAsk, bool) {
	ask, asked := ceilings.lastAskOfEachAddress[address]
	delete(ceilings.lastAskOfEachAddress, address)

	return ask, asked
}

func (ceilings *AskCeilings) documentsPerSecondOf(documentsAsked int, spent time.Duration) float64 {
	return min(
		float64(documentsAsked)/max(spent, shortestSpentTime).Seconds(),
		float64(ceilings.mostDocuments)/ceilings.targetTime.Seconds(),
	)
}

func (ceilings *AskCeilings) addSample(
	ctx context.Context,
	address string,
	sampledDocumentsPerSecond float64,
) {
	documentsPerSecond := sampledDocumentsPerSecond
	if previousDocumentsPerSecond, sampled := ceilings.documentsPerSecondOfEachAddress[address]; sampled {
		documentsPerSecond = weightOfTheNewSample*sampledDocumentsPerSecond +
			(1-weightOfTheNewSample)*previousDocumentsPerSecond
	}
	ceilings.documentsPerSecondOfEachAddress[address] = documentsPerSecond
	ceiling := ceilings.ceilingAt(documentsPerSecond)
	if ceiling < ceilings.mostDocuments {
		ceilings.observer.AskCeilingSet(ctx, address, documentsPerSecond, ceiling)
	}
}

func (ceilings *AskCeilings) AskCancelled(
	ctx context.Context,
	address string,
	spent time.Duration,
) {
	ceilings.mutex.Lock()
	defer ceilings.mutex.Unlock()

	ask, asked := ceilings.lastAskTakenFrom(address)
	if !asked {
		return
	}
	provenDocumentsPerSecond := ceilings.documentsPerSecondOf(ask.documentsAsked, spent)
	documentsPerSecond, sampled := ceilings.documentsPerSecondOfEachAddress[address]
	if sampled && provenDocumentsPerSecond >= documentsPerSecond {
		return
	}
	ceilings.addSample(ctx, address, provenDocumentsPerSecond)
}

func (ceilings *AskCeilings) AskFailed(ctx context.Context, address string) {
	ceilings.mutex.Lock()
	defer ceilings.mutex.Unlock()

	if _, asked := ceilings.lastAskTakenFrom(address); !asked {
		return
	}
	ceilings.addSample(ctx, address, 0)
}
