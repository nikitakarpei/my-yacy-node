// Package askceilings hands the outcome of each URL metadata call, and the
// time it took, to the URL metadata ask ceilings of the peers.
package askceilings

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
	_ int,
	spent time.Duration,
) {
	peerCalls.ceilings.AskAnswered(ctx, address, spent)
}

func (PeerCallCeilings) PeerSearchedDocuments(
	context.Context, string, int, int, time.Duration,
) {
}

func (peerCalls PeerCallCeilings) PeerRefused(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
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

func (peerCalls PeerCallCeilings) PeerUnreachable(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	_ error,
	_ time.Duration,
) {
	peerCalls.urlMetadataCallFailed(ctx, address, askedFor)
}

func (peerCalls PeerCallCeilings) PeerAnswerUnreadable(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	_ error,
	_ time.Duration,
) {
	peerCalls.urlMetadataCallFailed(ctx, address, askedFor)
}

func (peerCalls PeerCallCeilings) PeerCallCancelled(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	spent time.Duration,
) {
	if askedFor != peerasks.URLMetadata {
		return
	}
	peerCalls.ceilings.AskCancelled(ctx, address, spent)
}
