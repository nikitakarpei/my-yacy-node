package peerannouncement

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/bootstrap"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
)

const announceHelloPeerCount = 30

type ReportedHostObserver interface {
	ObserveReportedHosts(ctx context.Context, reportedHosts []yacymodel.Host)
}

type peerRoster interface {
	Discover(ctx context.Context, seeds ...yacymodel.Seed)
	ConfirmFirsthandSeed(ctx context.Context, seed yacymodel.Seed, answeredBaseURL peerwire.BaseURL)
	ConfirmUnreachable(ctx context.Context, peer yacymodel.Hash)
	ReachablePeers(ctx context.Context) []yacymodel.Seed
	UnreachablePeerHashes(ctx context.Context, limit int) []yacymodel.Hash
	RecordOf(ctx context.Context, peer yacymodel.Hash) (peerroster.PeerRecord, bool)
}

// TECHDEBT: vocabulary — one fact spelled four ways here: regreet, announce, contact, greet
type announcer struct {
	interval             time.Duration
	reachableCap         int
	contactConcurrency   int
	self                 SelfSeed
	reportedHostObserver ReportedHostObserver
	seeds                bootstrap.SeedSource
	roster               peerRoster
	greeter              peerGreeter
	observer             AnnouncementObserver
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
	announcedSeed := a.self.SelfSeed(ctx)
	contactReplies := a.contactPeers(ctx, announcedSeed)
	a.reportedHostObserver.ObserveReportedHosts(ctx, reportedHostsOf(contactReplies))
	if readdressedSeed, readdressed := a.readdressedSeedSince(ctx, announcedSeed); readdressed {
		a.contactPeers(ctx, readdressedSeed)
	}
}

func (a *announcer) contactPeers(ctx context.Context, self yacymodel.Seed) []contactReply {
	reachablePeers := a.roster.ReachablePeers(ctx)
	peerHashes := yacymodel.PeerHashesOf(reachablePeers)
	if deficit := a.reachableCap - len(reachablePeers); deficit > 0 {
		peerHashes = append(
			peerHashes,
			a.roster.UnreachablePeerHashes(ctx, deficit)...,
		)
	}

	contactReplies := a.contactAll(ctx, self, peerHashes)
	a.observer.ObservePeerContacts(
		amountOfPeersPerContactOutcome(contactReplies),
		amountOfReachablePeersPerScheme(contactReplies),
	)

	return contactReplies
}

func (a *announcer) readdressedSeedSince(
	ctx context.Context,
	announcedSeed yacymodel.Seed,
) (yacymodel.Seed, bool) {
	currentSeed := a.self.SelfSeed(ctx)

	return currentSeed, currentSeed.PrimaryHost != announcedSeed.PrimaryHost
}

func (a *announcer) contactAll(
	ctx context.Context,
	self yacymodel.Seed,
	peerHashes []yacymodel.Hash,
) []contactReply {
	concurrency := max(a.contactConcurrency, 1)
	slots := make(chan struct{}, concurrency)
	replies := make(chan contactReply, len(peerHashes))

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
			defer func() { <-slots }()
			replies <- a.contactOne(ctx, self, peerHash)
		}(peerHash)
	}

	contactReplies := make([]contactReply, 0, pending)
	for range pending {
		contactReplies = append(contactReplies, <-replies)
	}

	return contactReplies
}

func (a *announcer) contactOne(
	ctx context.Context,
	self yacymodel.Seed,
	peerHash yacymodel.Hash,
) contactReply {
	record, found := a.roster.RecordOf(ctx, peerHash)
	if !found {
		return contactReply{outcome: PeerDidNotAnswer}
	}

	result, answered := a.greeter.Greet(ctx, record, self, announceHelloPeerCount)
	if !answered {
		a.roster.ConfirmUnreachable(ctx, peerHash)
		slog.WarnContext(ctx, "peer greet failed", slog.String("peer", peerHash.String()))

		return contactReply{outcome: PeerDidNotAnswer}
	}
	reportedPeerType, present := result.ObservedPeerType.Get()
	if !present || !confirmsOurNetwork(reportedPeerType) {
		a.roster.ConfirmUnreachable(ctx, peerHash)
		slog.WarnContext(
			ctx,
			"peer did not confirm our network",
			slog.String("peer", peerHash.String()),
			slog.String("baseUrl", result.BaseURL.String()),
		)

		return contactReply{outcome: contactOutcomeFromObservedPeerType(result.ObservedPeerType)}
	}
	if reportedPeerType == yacymodel.PeerJunior {
		slog.WarnContext(
			ctx,
			"peer reported us as junior",
			slog.String("peer", peerHash.String()),
			slog.String("baseUrl", result.BaseURL.String()),
			slog.String("reportedAddress", result.ReportedHost.OrElse(yacymodel.Host{}).String()),
		)
	}

	a.roster.ConfirmFirsthandSeed(ctx, result.RespondingSeed, result.BaseURL)
	a.roster.Discover(ctx, result.KnownSeeds...)

	return contactReply{
		outcome:         contactOutcomeFromObservedPeerType(result.ObservedPeerType),
		reportedHost:    credibleReportedHostOf(result),
		reachableScheme: yacymodel.Some(result.BaseURL.Scheme),
	}
}

func credibleReportedHostOf(result greetResult) yacymodel.Optional[yacymodel.Host] {
	if _, isAddr := result.BaseURL.Host.Addr(); !isAddr {
		return yacymodel.None[yacymodel.Host]()
	}

	return result.ReportedHost
}

func confirmsOurNetwork(reportedPeerType yacymodel.PeerType) bool {
	switch reportedPeerType {
	case yacymodel.PeerJunior, yacymodel.PeerSenior, yacymodel.PeerPrincipal:
		return true
	default:
		return false
	}
}
