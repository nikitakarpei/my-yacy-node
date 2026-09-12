package peercallwire

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type PeerCallObserver interface {
	PeerAnsweredMatchedItems(
		ctx context.Context,
		address string,
		amountOfMatchedItems int,
		spent time.Duration,
	)
	PeerAnsweredURLMetadata(
		ctx context.Context,
		address string,
		amountOfDescribedDocuments int,
		spent time.Duration,
	)
	PeerAnsweredHeldDocuments(
		ctx context.Context,
		address string,
		amountOfDocuments int,
		spent time.Duration,
	)
	PeerRefused(
		ctx context.Context,
		address string,
		askedFor peerasks.AskedFor,
		status int,
		spent time.Duration,
	)
	PeerUnreachable(
		ctx context.Context,
		address string,
		askedFor peerasks.AskedFor,
		cause error,
		spent time.Duration,
	)
	PeerAnswerUnreadable(
		ctx context.Context,
		address string,
		askedFor peerasks.AskedFor,
		cause error,
		spent time.Duration,
	)
}

type PeerCallObservers []PeerCallObserver

func (observers PeerCallObservers) PeerAnsweredMatchedItems(
	ctx context.Context,
	address string,
	amountOfMatchedItems int,
	spent time.Duration,
) {
	for _, observer := range observers {
		observer.PeerAnsweredMatchedItems(ctx, address, amountOfMatchedItems, spent)
	}
}

func (observers PeerCallObservers) PeerAnsweredURLMetadata(
	ctx context.Context,
	address string,
	amountOfDescribedDocuments int,
	spent time.Duration,
) {
	for _, observer := range observers {
		observer.PeerAnsweredURLMetadata(ctx, address, amountOfDescribedDocuments, spent)
	}
}

func (observers PeerCallObservers) PeerAnsweredHeldDocuments(
	ctx context.Context,
	address string,
	amountOfDocuments int,
	spent time.Duration,
) {
	for _, observer := range observers {
		observer.PeerAnsweredHeldDocuments(ctx, address, amountOfDocuments, spent)
	}
}

func (observers PeerCallObservers) PeerRefused(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	status int,
	spent time.Duration,
) {
	for _, observer := range observers {
		observer.PeerRefused(ctx, address, askedFor, status, spent)
	}
}

func (observers PeerCallObservers) PeerUnreachable(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	cause error,
	spent time.Duration,
) {
	for _, observer := range observers {
		observer.PeerUnreachable(ctx, address, askedFor, cause, spent)
	}
}

func (observers PeerCallObservers) PeerAnswerUnreadable(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	cause error,
	spent time.Duration,
) {
	for _, observer := range observers {
		observer.PeerAnswerUnreadable(ctx, address, askedFor, cause, spent)
	}
}
