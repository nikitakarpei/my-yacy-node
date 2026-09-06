package peeradmission

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const maxKnownPeersPerHello = 100

const (
	msgForeignNetwork = "not in my network"
	msgAdmitted       = "ok"
)

type callerReachabilityProbe interface {
	IsReachable(
		ctx context.Context,
		caller yacymodel.Seed,
		self yacymodel.Hash,
		networkName string,
	) bool
}

type reachableRoster interface {
	MostRecentlyReachablePeers(ctx context.Context, limit int) []yacymodel.Seed
	ConfirmReachable(ctx context.Context, seed yacymodel.Seed)
}

type helloEndpoint struct {
	identity     nodeidentity.Identity
	status       RuntimeStatus
	now          func() time.Time
	probe        callerReachabilityProbe
	reachability reachableRoster
}

func (e helloEndpoint) Serve(
	ctx context.Context,
	req yacyproto.HelloRequest,
) (yacyproto.HelloResponse, error) {
	if !e.identity.NetworkMatches(req.NetworkName) {
		return e.foreignNetworkReply(ctx), nil
	}

	return e.sameNetworkReply(ctx, req), nil
}

func (e helloEndpoint) foreignNetworkReply(ctx context.Context) yacyproto.HelloResponse {
	slog.DebugContext(ctx, "hello refused, caller is on another network")

	return yacyproto.HelloResponse{
		YourIP:   httpguard.RemoteAddr(ctx),
		YourType: yacymodel.Some(yacymodel.PeerVirgin),
		MyTime:   e.now(),
		Message:  msgForeignNetwork,
	}
}

func (e helloEndpoint) sameNetworkReply(
	ctx context.Context,
	req yacyproto.HelloRequest,
) yacyproto.HelloResponse {
	seeds := append(
		[]yacymodel.Seed{e.status.SelfSeed(ctx)},
		e.knownPeers(ctx, req.Count)...,
	)

	slog.DebugContext(ctx, "hello served", slog.Int("seedCount", len(seeds)))

	return yacyproto.HelloResponse{
		YourIP:   httpguard.RemoteAddr(ctx),
		YourType: yacymodel.Some(e.classifyCaller(ctx, req.Seed)),
		MyTime:   e.now(),
		Message:  msgAdmitted,
		Seeds:    seeds,
	}
}

func (e helloEndpoint) classifyCaller(
	ctx context.Context,
	caller yacymodel.Seed,
) yacymodel.PeerType {
	if _, ok := caller.NetworkAddress(); !ok {
		return yacymodel.PeerJunior
	}

	if !e.probe.IsReachable(ctx, caller, e.status.SelfSeed(ctx).Hash, e.status.NetworkName(ctx)) {
		return yacymodel.PeerJunior
	}

	e.reachability.ConfirmReachable(ctx, caller)

	if caller.PeerType == yacymodel.PeerPrincipal {
		return yacymodel.PeerPrincipal
	}

	return yacymodel.PeerSenior
}

func (e helloEndpoint) knownPeers(ctx context.Context, count int) []yacymodel.Seed {
	return e.reachability.MostRecentlyReachablePeers(ctx, min(count, maxKnownPeersPerHello))
}
