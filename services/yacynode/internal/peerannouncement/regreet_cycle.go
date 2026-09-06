package peerannouncement

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/bootstrap"
)

const announceHelloPeerCount = 30

type peerRoster interface {
	Discover(ctx context.Context, seeds ...yacymodel.Seed)
	ConfirmReachable(ctx context.Context, seed yacymodel.Seed)
	ConfirmUnreachable(ctx context.Context, peer yacymodel.Hash)
	ReachablePeers(ctx context.Context) []yacymodel.Seed
	UnreachablePeerHashes(ctx context.Context, limit int) []yacymodel.Hash
	NetworkAddressOf(
		ctx context.Context,
		peer yacymodel.Hash,
	) (yacymodel.NetworkAddress, bool)
}

// TECHDEBT: vocabulary — one fact spelled four ways here: regreet, announce, contact, greet
type announcer struct {
	interval           time.Duration
	reachableCap       int
	contactConcurrency int
	self               SelfSeed
	seeds              bootstrap.SeedSource
	roster             peerRoster
	greeter            httpPeerGreeter
}

func (a *announcer) Run(ctx context.Context) {
	a.roster.Discover(ctx, a.seeds.Fetch(ctx)...)
	a.announce(ctx)

	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.announce(ctx)
		}
	}
}

func (a *announcer) announce(ctx context.Context) {
	self := a.self.SelfSeed(ctx)

	reachablePeers := a.roster.ReachablePeers(ctx)
	peerHashes := yacymodel.PeerHashesOf(reachablePeers)
	if deficit := a.reachableCap - len(reachablePeers); deficit > 0 {
		peerHashes = append(
			peerHashes,
			a.roster.UnreachablePeerHashes(ctx, deficit)...,
		)
	}

	a.contactAll(ctx, self, peerHashes)
}

func (a *announcer) contactAll(
	ctx context.Context,
	self yacymodel.Seed,
	peerHashes []yacymodel.Hash,
) {
	concurrency := max(a.contactConcurrency, 1)
	slots := make(chan struct{}, concurrency)
	done := make(chan struct{}, len(peerHashes))

	pending := 0
	for _, peerHash := range peerHashes {
		if peerHash == self.Hash {
			slog.DebugContext(
				ctx,
				"skipped self in contact targets",
				slog.String("peer", peerHash.String()),
			)

			continue
		}

		pending++
		slots <- struct{}{}
		go func(peerHash yacymodel.Hash) {
			defer func() { <-slots; done <- struct{}{} }()
			a.contactOne(ctx, self, peerHash)
		}(peerHash)
	}

	for range pending {
		<-done
	}
}

func (a *announcer) contactOne(
	ctx context.Context,
	self yacymodel.Seed,
	peerHash yacymodel.Hash,
) {
	networkAddress, found := a.roster.NetworkAddressOf(ctx, peerHash)
	if !found {
		return
	}

	result, err := a.greeter.Greet(ctx, networkAddress, self, announceHelloPeerCount)
	if err != nil {
		a.roster.ConfirmUnreachable(ctx, peerHash)
		slog.WarnContext(
			ctx,
			"peer greet failed",
			slog.String("peer", peerHash.String()),
			slog.String("endpoint", networkAddress.String()),
			slog.Any("error", err),
		)

		return
	}
	if result.RespondingSeed.Hash != peerHash {
		a.roster.ConfirmUnreachable(ctx, peerHash)
		slog.WarnContext(
			ctx,
			"peer greet identified a different peer",
			slog.String("peer", peerHash.String()),
			slog.String("respondingPeer", result.RespondingSeed.Hash.String()),
		)
	}
	if _, addressable := result.RespondingSeed.NetworkAddress(); !addressable {
		a.roster.ConfirmUnreachable(ctx, peerHash)
		slog.WarnContext(
			ctx,
			"peer greet returned an unaddressable seed",
			slog.String("peer", peerHash.String()),
		)

		return
	}
	reportedPeerType, present := result.ObservedPeerType.Get()
	if !present || !confirmsOurNetwork(reportedPeerType) {
		a.roster.ConfirmUnreachable(ctx, peerHash)
		slog.WarnContext(
			ctx,
			"peer did not confirm our network",
			slog.String("peer", peerHash.String()),
			slog.String("endpoint", networkAddress.String()),
		)

		return
	}
	if reportedPeerType == yacymodel.PeerJunior {
		slog.WarnContext(
			ctx,
			"peer reported us as junior",
			slog.String("peer", peerHash.String()),
			slog.String("endpoint", networkAddress.String()),
			slog.String("reportedAddress", result.ObservedAddress),
		)
	}

	a.roster.ConfirmReachable(ctx, result.RespondingSeed)
	a.roster.Discover(ctx, result.KnownSeeds...)
}

func confirmsOurNetwork(reportedPeerType yacymodel.PeerType) bool {
	switch reportedPeerType {
	case yacymodel.PeerJunior, yacymodel.PeerSenior, yacymodel.PeerPrincipal:
		return true
	default:
		return false
	}
}
