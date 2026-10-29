package peeradmission

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	backPingsAtOnce  = 16
	backPingBudget   = 13 * time.Second
	backPingSlotWait = 15 * time.Second
)

type peerRecords interface {
	RecordOf(ctx context.Context, peer yacymodel.Hash) (peerroster.PeerRecord, bool)
}

type callerBackPing struct {
	walker      peerwire.Walker
	peerRecords peerRecords
	clock       Clock
	slots       chan struct{}
}

func newCallerBackPing(
	walker peerwire.Walker,
	records peerRecords,
	clock Clock,
) callerBackPing {
	return callerBackPing{
		walker:      walker,
		peerRecords: records,
		clock:       clock,
		slots:       make(chan struct{}, backPingsAtOnce),
	}
}

var _ callerReachabilityProbe = callerBackPing{}

func (p callerBackPing) Reach(
	ctx context.Context,
	caller yacymodel.Seed,
	addresses yacymodel.PeerAddresses,
	self yacymodel.Hash,
	networkName string,
) (peerwire.BaseURL, bool) {
	if !p.takeSlot(ctx) {
		slog.WarnContext(ctx, "caller back-ping skipped, no back-ping slot freed in time",
			slog.String("peer", caller.Hash.String()),
		)

		return peerwire.BaseURL{}, false
	}
	defer func() { <-p.slots }()

	query := yacyproto.QueryRequest{
		NetworkName: networkName,
		YouAre:      caller.Hash,
		Iam:         self,
		Object:      yacyproto.ObjectRWICount,
	}
	identity := caller.IdentityIn(networkName)
	answer, answered := p.walker.TryInTurn(
		ctx,
		peerwire.Request{
			Callee: peerwire.Callee{Hash: caller.Hash, Identity: identity},
			Path:   yacyproto.PathQuery,
			Form:   query.Form(),
		},
		p.baseURLsAnsweredFirst(ctx, caller.Hash, peerwire.BaseURLsFrom(addresses, identity)),
		isRWICountReply,
		backPingBudget,
	)

	return answer.BaseURL, answered
}

func (p callerBackPing) takeSlot(ctx context.Context) bool {
	select {
	case p.slots <- struct{}{}:
		return true
	default:
	}

	waitEnded := make(chan struct{})
	stop := p.clock.After(backPingSlotWait, func() { close(waitEnded) })
	defer stop()

	select {
	case p.slots <- struct{}{}:
		return true
	case <-waitEnded:
		return false
	case <-ctx.Done():
		return false
	}
}

func (p callerBackPing) baseURLsAnsweredFirst(
	ctx context.Context,
	caller yacymodel.Hash,
	baseURLs []peerwire.BaseURL,
) []peerwire.BaseURL {
	record, _ := p.peerRecords.RecordOf(ctx, caller)
	answeredBaseURL, found := record.AnsweredBaseURL.Get()
	if !found || !slices.Contains(baseURLs, answeredBaseURL) {
		return baseURLs
	}

	return append(
		[]peerwire.BaseURL{answeredBaseURL},
		slices.DeleteFunc(
			baseURLs,
			func(baseURL peerwire.BaseURL) bool { return baseURL == answeredBaseURL },
		)...,
	)
}

func isRWICountReply(fields yacyproto.ResponseFields) bool {
	if fields[yacyproto.FieldResponse] == "" {
		return false
	}
	resp, err := yacyproto.ParseQueryResponse(fields)

	return err == nil && resp.Response != yacyproto.QueryResponseRejected
}
