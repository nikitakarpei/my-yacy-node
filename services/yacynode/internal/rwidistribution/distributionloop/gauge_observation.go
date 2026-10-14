package distributionloop

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
)

const gaugeObservationInterval = 15 * time.Second

type gaugeObservation struct {
	vault           *vault.Vault
	undelivered     *undeliveredpostings.Postings
	roster          ReachablePeers
	clock           Clock
	walk            *arcWalk
	backlogObserver BacklogObserver
	dhtRingObserver DHTRingObserver
}

func (g *gaugeObservation) observeUntilDone(ctx context.Context) {
	for ctx.Err() == nil {
		g.observeBacklog(ctx)
		g.observeArcs()
		g.observePeersAcceptingRemoteIndex(ctx)
		sleepOn(ctx, g.clock, gaugeObservationInterval)
	}
}

func (g *gaugeObservation) observeBacklog(ctx context.Context) {
	amountOfUndeliveredPostings, err := g.amountOfUndeliveredPostings(ctx)
	if err != nil {
		slog.WarnContext(ctx, "distribution backlog not observed", slog.Any("error", err))

		return
	}
	g.backlogObserver.ObserveUndeliveredPostings(amountOfUndeliveredPostings)
}

func (g *gaugeObservation) observeArcs() {
	amountPerState := make(map[string]int, len(arcStates))
	for state, amount := range g.walk.amountOfArcsPerState() {
		amountPerState[string(state)] = amount
	}
	g.backlogObserver.ObserveArcs(amountPerState)
}

func (g *gaugeObservation) amountOfUndeliveredPostings(ctx context.Context) (int, error) {
	var amountOfPostings int
	err := g.vault.View(ctx, func(tx *vault.Txn) error {
		var err error
		amountOfPostings, err = g.undelivered.AmountOfPostings(tx)

		return err
	})

	return amountOfPostings, err
}

func (g *gaugeObservation) observePeersAcceptingRemoteIndex(ctx context.Context) {
	g.dhtRingObserver.ObservePeersAcceptingRemoteIndexPerDHTRingSector(
		yacymodel.SeedsPerDHTRingSector(
			peersAcceptingRemoteIndexAmong(g.roster.ReachablePeers(ctx)),
		),
	)
}

func peersAcceptingRemoteIndexAmong(peers []yacymodel.Seed) []yacymodel.Seed {
	acceptingPeers := make([]yacymodel.Seed, 0, len(peers))
	for _, seed := range peers {
		if seed.AcceptsRemoteIndex() {
			acceptingPeers = append(acceptingPeers, seed)
		}
	}

	return acceptingPeers
}
