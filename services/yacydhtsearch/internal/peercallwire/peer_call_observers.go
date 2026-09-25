package peercallwire

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type PeerCallObserver interface {
	PeerCallWaitsForASlot(
		ctx context.Context,
		address string,
		askedFor peerasks.AskedFor,
	)
	PeerCallTookASlot(
		ctx context.Context,
		address string,
		askedFor peerasks.AskedFor,
		waited time.Duration,
	)
	PeerAnsweredURLMetadata(
		ctx context.Context,
		address string,
		amountOfDocumentsAsked int,
		amountOfDescribedDocuments int,
		spent time.Duration,
	)
	PeerSearchedDocuments(
		ctx context.Context,
		address string,
		amountOfDocumentsInTheAbstract int,
		amountOfMatchedDocuments int,
		spent time.Duration,
	)
	PeerListedTheAbstract(
		ctx context.Context,
		address string,
		amountOfDocumentsInTheAbstract int,
		spent time.Duration,
	)
	PeerRefused(
		ctx context.Context,
		address string,
		askedFor peerasks.AskedFor,
		amountOfDocumentsAsked int,
		status int,
		spent time.Duration,
	)
	PeerUnreachable(
		ctx context.Context,
		address string,
		askedFor peerasks.AskedFor,
		amountOfDocumentsAsked int,
		cause error,
		spent time.Duration,
	)
	PeerAnswerUnreadable(
		ctx context.Context,
		address string,
		askedFor peerasks.AskedFor,
		amountOfDocumentsAsked int,
		cause error,
		spent time.Duration,
	)
	PeerHeadersLate(
		ctx context.Context,
		address string,
		askedFor peerasks.AskedFor,
		amountOfDocumentsAsked int,
		spent time.Duration,
	)
	PeerCallCancelled(
		ctx context.Context,
		address string,
		askedFor peerasks.AskedFor,
		amountOfDocumentsAsked int,
		spent time.Duration,
	)
}

type PeerCallObservers []PeerCallObserver

func (observers PeerCallObservers) PeerCallWaitsForASlot(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
) {
	for _, observer := range observers {
		observer.PeerCallWaitsForASlot(ctx, address, askedFor)
	}
}

func (observers PeerCallObservers) PeerCallTookASlot(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	waited time.Duration,
) {
	for _, observer := range observers {
		observer.PeerCallTookASlot(ctx, address, askedFor, waited)
	}
}

func (observers PeerCallObservers) PeerAnsweredURLMetadata(
	ctx context.Context,
	address string,
	amountOfDocumentsAsked int,
	amountOfDescribedDocuments int,
	spent time.Duration,
) {
	for _, observer := range observers {
		observer.PeerAnsweredURLMetadata(
			ctx, address, amountOfDocumentsAsked, amountOfDescribedDocuments, spent,
		)
	}
}

func (observers PeerCallObservers) PeerSearchedDocuments(
	ctx context.Context,
	address string,
	amountOfDocumentsInTheAbstract int,
	amountOfMatchedDocuments int,
	spent time.Duration,
) {
	for _, observer := range observers {
		observer.PeerSearchedDocuments(
			ctx, address, amountOfDocumentsInTheAbstract, amountOfMatchedDocuments, spent,
		)
	}
}

func (observers PeerCallObservers) PeerListedTheAbstract(
	ctx context.Context,
	address string,
	amountOfDocumentsInTheAbstract int,
	spent time.Duration,
) {
	for _, observer := range observers {
		observer.PeerListedTheAbstract(ctx, address, amountOfDocumentsInTheAbstract, spent)
	}
}

//nolint:revive // argument-limit: an outcome names its peer, ask, documents asked, failure and time
func (observers PeerCallObservers) PeerRefused(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	amountOfDocumentsAsked int,
	status int,
	spent time.Duration,
) {
	for _, observer := range observers {
		observer.PeerRefused(ctx, address, askedFor, amountOfDocumentsAsked, status, spent)
	}
}

//nolint:revive // argument-limit: an outcome names its peer, ask, documents asked, failure and time
func (observers PeerCallObservers) PeerUnreachable(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	amountOfDocumentsAsked int,
	cause error,
	spent time.Duration,
) {
	for _, observer := range observers {
		observer.PeerUnreachable(ctx, address, askedFor, amountOfDocumentsAsked, cause, spent)
	}
}

//nolint:revive // argument-limit: an outcome names its peer, ask, documents asked, failure and time
func (observers PeerCallObservers) PeerAnswerUnreadable(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	amountOfDocumentsAsked int,
	cause error,
	spent time.Duration,
) {
	for _, observer := range observers {
		observer.PeerAnswerUnreadable(ctx, address, askedFor, amountOfDocumentsAsked, cause, spent)
	}
}

func (observers PeerCallObservers) PeerHeadersLate(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	amountOfDocumentsAsked int,
	spent time.Duration,
) {
	for _, observer := range observers {
		observer.PeerHeadersLate(ctx, address, askedFor, amountOfDocumentsAsked, spent)
	}
}

func (observers PeerCallObservers) PeerCallCancelled(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	amountOfDocumentsAsked int,
	spent time.Duration,
) {
	for _, observer := range observers {
		observer.PeerCallCancelled(ctx, address, askedFor, amountOfDocumentsAsked, spent)
	}
}
