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

type answeredBaseURLs interface {
	AnsweredBaseURLOf(ctx context.Context, peer yacymodel.Hash) (peerwire.BaseURL, bool)
}

type callerBackPing struct {
	exchange         peerwire.FormExchange
	answeredBaseURLs answeredBaseURLs
	slots            chan struct{}
}

func newCallerBackPing(exchange peerwire.FormExchange, baseURLs answeredBaseURLs) callerBackPing {
	return callerBackPing{
		exchange:         exchange,
		answeredBaseURLs: baseURLs,
		slots:            make(chan struct{}, backPingsAtOnce),
	}
}

var _ callerReachabilityProbe = callerBackPing{}

func (p callerBackPing) Reach(
	ctx context.Context,
	caller yacymodel.Hash,
	addresses yacymodel.PeerAddresses,
	self yacymodel.Hash,
	networkName string,
) (peerwire.BaseURL, bool) {
	select {
	case p.slots <- struct{}{}:
		defer func() { <-p.slots }()
	default:
		slog.WarnContext(ctx, "caller back-ping skipped, too many back-pings running",
			slog.String("peer", caller.String()),
		)

		return peerwire.BaseURL{}, false
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
		p.baseURLsFor(ctx, caller, addresses),
		isRWICountReply,
		backPingBudget,
	)

	return answer.BaseURL, answered
}

func (p callerBackPing) baseURLsFor(
	ctx context.Context,
	caller yacymodel.Hash,
	addresses yacymodel.PeerAddresses,
) []peerwire.BaseURL {
	baseURLs := peerwire.BaseURLsFrom(addresses)
	answeredBaseURL, found := p.answeredBaseURLs.AnsweredBaseURLOf(ctx, caller)
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
