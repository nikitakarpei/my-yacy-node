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
	ConfirmReachable(ctx context.Context, peer yacymodel.Hash)
	ConfirmUnreachable(ctx context.Context, peer yacymodel.Hash)
	ReachablePeers(ctx context.Context) []yacymodel.Seed
	UnreachablePeers(ctx context.Context, limit int) []yacymodel.Seed
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

	targets := a.roster.ReachablePeers(ctx)
	if deficit := a.reachableCap - len(targets); deficit > 0 {
		targets = append(targets, a.roster.UnreachablePeers(ctx, deficit)...)
	}

	a.contactAll(ctx, self, targets)
}

// TECHDEBT: abstraction — contactAll picks targets and also runs the fan-out concurrency
func (a *announcer) contactAll(ctx context.Context, self yacymodel.Seed, targets []yacymodel.Seed) {
	concurrency := max(a.contactConcurrency, 1)
	slots := make(chan struct{}, concurrency)
	done := make(chan struct{}, len(targets))

	pending := 0
	for _, target := range targets {
		if target.Hash == self.Hash {
			slog.DebugContext(
				ctx,
				"skipped self in contact targets",
				slog.String("peer", target.Hash.String()),
			)

			continue
		}

		endpoint, ok := target.NetworkAddress()
		if !ok {
			continue
		}

		pending++
		slots <- struct{}{}
		go func(target yacymodel.Seed, endpoint string) {
			defer func() { <-slots; done <- struct{}{} }()
			a.contactOne(ctx, self, target, endpoint)
		}(target, endpoint)
	}

	for range pending {
		<-done
	}
}

func (a *announcer) contactOne(
	ctx context.Context,
	self, target yacymodel.Seed,
	endpoint string,
) {
	result, err := a.greeter.Greet(ctx, endpoint, self, announceHelloPeerCount)
	if err != nil {
		a.roster.ConfirmUnreachable(ctx, target.Hash)
		slog.WarnContext(
			ctx,
			"peer greet failed",
			slog.String("peer", target.Hash.String()),
			slog.String("endpoint", endpoint),
			slog.Any("error", err),
		)

		return
	}
	yourType, confirmed := result.YourType.Get()
	if !confirmed {
		a.roster.ConfirmUnreachable(ctx, target.Hash)
		slog.WarnContext(
			ctx,
			"peer did not confirm our network",
			slog.String("peer", target.Hash.String()),
			slog.String("endpoint", endpoint),
		)

		return
	}
	if yourType == yacymodel.PeerJunior {
		slog.WarnContext(
			ctx,
			"peer reported us as junior",
			slog.String("peer", target.Hash.String()),
			slog.String("endpoint", endpoint),
			slog.String("reportedAddress", result.YourIP),
		)
	}

	a.roster.ConfirmReachable(ctx, target.Hash)
	a.roster.Discover(ctx, result.Known...)
}
