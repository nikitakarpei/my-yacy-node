// Package urlmetadataaskceilings sizes, for each peer address, the most
// documents one URL metadata ask may name, from a moving average of the peer's
// pace: the time it spends per document it describes.
package urlmetadataaskceilings

import (
	"context"
	"math"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const weightOfTheNewSample = 0.5

type AskCeilings struct {
	mostDocuments  int
	leastDocuments int
	targetTime     time.Duration
	peerPaces      PeerPaces
	observer       AskCeilingObserver
}

func New(
	mostDocuments int,
	leastDocuments int,
	targetTime time.Duration,
	peerPaces PeerPaces,
	observer AskCeilingObserver,
) *AskCeilings {
	return &AskCeilings{
		mostDocuments:  mostDocuments,
		leastDocuments: min(leastDocuments, mostDocuments),
		targetTime:     targetTime,
		peerPaces:      peerPaces,
		observer:       observer,
	}
}

func (ceilings *AskCeilings) CeilingOf(ctx context.Context, address string) int {
	pace, known := ceilings.peerPaces.
		Read(ctx, address).Get()
	if !known {
		return ceilings.mostDocuments
	}

	return ceilings.ceilingFrom(pace)
}

func (ceilings *AskCeilings) ceilingFrom(pace time.Duration) int {
	documentsWithinTheTargetTime := math.Round(
		float64(ceilings.targetTime) / float64(pace),
	)

	return min(
		max(int(documentsWithinTheTargetTime), ceilings.leastDocuments),
		ceilings.mostDocuments,
	)
}

func (ceilings *AskCeilings) AskAnswered(
	ctx context.Context,
	address string,
	amountOfDocumentsAsked int,
	spent time.Duration,
) {
	if amountOfDocumentsAsked < ceilings.CeilingOf(ctx, address) {
		return
	}
	ceilings.blend(ctx, address, ceilings.paceFrom(amountOfDocumentsAsked, spent))
}

func (ceilings *AskCeilings) paceFrom(
	amountOfDocumentsAsked int,
	spent time.Duration,
) time.Duration {
	pace := spent / time.Duration(amountOfDocumentsAsked)

	return min(
		max(pace, ceilings.shortestPace()),
		ceilings.longestPace(),
	)
}

func (ceilings *AskCeilings) shortestPace() time.Duration {
	return ceilings.targetTime / time.Duration(ceilings.mostDocuments)
}

func (ceilings *AskCeilings) longestPace() time.Duration {
	return ceilings.targetTime / time.Duration(ceilings.leastDocuments)
}

func (ceilings *AskCeilings) blend(ctx context.Context, address string, sample time.Duration) {
	ceilings.update(
		ctx,
		address,
		func(pace yacymodel.Optional[time.Duration]) time.Duration {
			knownPace, known := pace.Get()
			if !known {
				return sample
			}

			return knownPace +
				time.Duration(weightOfTheNewSample*float64(sample-knownPace))
		},
	)
}

func (ceilings *AskCeilings) update(
	ctx context.Context,
	address string,
	updated func(pace yacymodel.Optional[time.Duration]) time.Duration,
) {
	pace := ceilings.peerPaces.Update(ctx, address, updated)
	ceilings.observer.AskCeilingSet(
		ctx, address, pace, ceilings.ceilingFrom(pace),
	)
}

func (ceilings *AskCeilings) AskCancelled(
	ctx context.Context,
	address string,
	amountOfDocumentsAsked int,
	spent time.Duration,
) {
	provenPace := ceilings.paceFrom(amountOfDocumentsAsked, spent)
	pace, known := ceilings.peerPaces.
		Read(ctx, address).Get()
	if known && provenPace <= pace {
		return
	}
	ceilings.blend(ctx, address, provenPace)
}

func (ceilings *AskCeilings) AskFailed(ctx context.Context, address string) {
	longestPace := ceilings.longestPace()
	ceilings.update(
		ctx,
		address,
		func(pace yacymodel.Optional[time.Duration]) time.Duration {
			knownPace, known := pace.Get()
			if !known {
				return longestPace
			}

			return min(2*knownPace, longestPace)
		},
	)
}
