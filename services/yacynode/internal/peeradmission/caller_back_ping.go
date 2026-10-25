package peeradmission

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	backPingsAtOnce = 16
	backPingBudget  = 13 * time.Second
)

type answeredOrigins interface {
	AnsweredOriginOf(ctx context.Context, peer yacymodel.Hash) (peerwire.Origin, bool)
}

type callerBackPing struct {
	exchange        peerwire.FormExchange
	answeredOrigins answeredOrigins
	slots           chan struct{}
}

func newCallerBackPing(exchange peerwire.FormExchange, origins answeredOrigins) callerBackPing {
	return callerBackPing{
		exchange:        exchange,
		answeredOrigins: origins,
		slots:           make(chan struct{}, backPingsAtOnce),
	}
}

var _ callerReachabilityProbe = callerBackPing{}

func (p callerBackPing) Reach(
	ctx context.Context,
	caller yacymodel.Hash,
	addresses yacymodel.PeerAddresses,
	self yacymodel.Hash,
	networkName string,
) (peerwire.Origin, bool) {
	select {
	case p.slots <- struct{}{}:
		defer func() { <-p.slots }()
	default:
		slog.WarnContext(ctx, "caller back-ping skipped, too many back-pings running",
			slog.String("peer", caller.String()),
		)

		return peerwire.Origin{}, false
	}

	query := yacyproto.QueryRequest{
		NetworkName: networkName,
		YouAre:      caller,
		Iam:         self,
		Object:      yacyproto.ObjectRWICount,
	}
	answer, answered := p.exchange.Exchange(
		ctx,
		peerwire.Request{Peer: caller, Path: yacyproto.PathQuery, Form: query.Form()},
		p.originsFor(ctx, caller, addresses),
		isRWICountReply,
		backPingBudget,
	)

	return answer.Origin, answered
}

func (p callerBackPing) originsFor(
	ctx context.Context,
	caller yacymodel.Hash,
	addresses yacymodel.PeerAddresses,
) []peerwire.Origin {
	origins := peerwire.OriginsFrom(addresses)
	answeredOrigin, found := p.answeredOrigins.AnsweredOriginOf(ctx, caller)
	if !found || !slices.Contains(origins, answeredOrigin) {
		return origins
	}

	return append(
		[]peerwire.Origin{answeredOrigin},
		slices.DeleteFunc(
			origins,
			func(origin peerwire.Origin) bool { return origin == answeredOrigin },
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
