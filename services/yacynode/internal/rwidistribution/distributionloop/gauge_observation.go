package distributionloop

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
)

const gaugeObservationInterval = 15 * time.Second

type gaugeObservation struct {
	vault           *vault.Vault
	offers          *wordpartitionoffer.WordPartitionOffers
	undelivered     *undeliveredpostings.UndeliveredPostings
	holders         *wordpartitionholders.Holders
	roster          ReachablePeers
	clock           Clock
	backlogObserver BacklogObserver
}

func (g *gaugeObservation) observeUntilDone(ctx context.Context) {
	for ctx.Err() == nil {
		g.observeBacklog(ctx)
		g.offers.ReportPeersAcceptingRemoteIndex(g.roster.ReachablePeers(ctx))
		sleepOn(ctx, g.clock, gaugeObservationInterval)
	}
}

func (g *gaugeObservation) observeBacklog(ctx context.Context) {
	backlog, err := g.backlogNow(ctx)
	if err != nil {
		slog.WarnContext(ctx, "distribution backlog not observed", slog.Any("error", err))

		return
	}
	g.backlogObserver.ObserveUndeliveredPostings(backlog.amountOfUndeliveredPostings)
	g.backlogObserver.ObserveLongestRefreshLateness(backlog.longestRefreshLateness)
}

type backlog struct {
	amountOfUndeliveredPostings int
	longestRefreshLateness      time.Duration
}

func (g *gaugeObservation) backlogNow(ctx context.Context) (backlog, error) {
	var current backlog
	err := g.vault.View(ctx, func(tx *vault.Txn) error {
		var err error
		current.amountOfUndeliveredPostings, err = g.undelivered.AmountOfPostings(tx)
		if err != nil {
			return err
		}
		current.longestRefreshLateness, err = g.longestRefreshLatenessIn(tx)

		return err
	})

	return current, err
}

func (g *gaugeObservation) longestRefreshLatenessIn(tx *vault.Txn) (time.Duration, error) {
	earliestRefreshAt, found, err := g.holders.EarliestRefreshAt(tx)
	if err != nil || !found {
		return 0, err
	}

	return max(g.clock.Now().Sub(earliestRefreshAt), 0), nil
}
