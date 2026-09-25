// Package urlmetadataaskceilings hands the outcome and time of each URL
// metadata call to the URL metadata ask ceilings of the peers.
package urlmetadataaskceilings

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/urlmetadataaskceilings"
)

type PeerCallCeilings struct {
	ceilings *urlmetadataaskceilings.AskCeilings
}

func New(ceilings *urlmetadataaskceilings.AskCeilings) PeerCallCeilings {
	return PeerCallCeilings{ceilings: ceilings}
}

func (PeerCallCeilings) PeerCallWaitsForASlot(context.Context, string, peerasks.AskedFor) {}

func (PeerCallCeilings) PeerCallTookASlot(
	context.Context, string, peerasks.AskedFor, time.Duration,
) {
}

func (peerCalls PeerCallCeilings) PeerAnsweredURLMetadata(
	ctx context.Context,
	address string,
	amountOfDocumentsAsked int,
	_ int,
	spent time.Duration,
) {
	peerCalls.ceilings.AskAnswered(ctx, address, amountOfDocumentsAsked, spent)
}

func (PeerCallCeilings) PeerSearchedDocuments(
	context.Context, string, int, int, time.Duration,
) {
}

//nolint:revive // argument-limit: an outcome names its peer, ask, documents asked, failure and time
func (peerCalls PeerCallCeilings) PeerRefused(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	_ int,
	_ int,
	_ time.Duration,
) {
	peerCalls.urlMetadataCallFailed(ctx, address, askedFor)
}

func (peerCalls PeerCallCeilings) urlMetadataCallFailed(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
) {
	if askedFor != peerasks.URLMetadata {
		return
	}
	peerCalls.ceilings.AskFailed(ctx, address)
}

//nolint:revive // argument-limit: an outcome names its peer, ask, documents asked, failure and time
func (peerCalls PeerCallCeilings) PeerUnreachable(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	_ int,
	_ error,
	_ time.Duration,
) {
	peerCalls.urlMetadataCallFailed(ctx, address, askedFor)
}

//nolint:revive // argument-limit: an outcome names its peer, ask, documents asked, failure and time
func (peerCalls PeerCallCeilings) PeerAnswerUnreadable(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	_ int,
	_ error,
	_ time.Duration,
) {
	peerCalls.urlMetadataCallFailed(ctx, address, askedFor)
}

func (peerCalls PeerCallCeilings) PeerHeadersLate(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	_ int,
	_ time.Duration,
) {
	peerCalls.urlMetadataCallFailed(ctx, address, askedFor)
}

func (peerCalls PeerCallCeilings) PeerCallCancelled(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	amountOfDocumentsAsked int,
	spent time.Duration,
) {
	if askedFor != peerasks.URLMetadata {
		return
	}
	peerCalls.ceilings.AskCancelled(ctx, address, amountOfDocumentsAsked, spent)
}
