// Package peerannouncement greets known peers on an interval: it announces this
// node to them and reports their reachability to the peer roster. It owns no peer
// data — it discovers candidates from the seed source on a cold start, reads
// targets from the roster, and writes reachability observations back.
package peerannouncement

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/bootstrap"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
)

type SelfSeed interface {
	SelfSeed(ctx context.Context) yacymodel.Seed
}

type Announcer interface {
	Run(ctx context.Context)
}

type Config struct {
	NetworkName        string
	Interval           time.Duration
	ReachableCap       int
	ContactConcurrency int
}

//nolint:revive // argument-limit: independently-meaningful collaborators, not configuration to bundle further
func New(
	cfg Config,
	exchange peerwire.FormExchange,
	self SelfSeed,
	reportedHostObserver ReportedHostObserver,
	seeds bootstrap.SeedSource,
	roster peerRoster,
	observer AnnouncementObserver,
) Announcer {
	return &announcer{
		interval:             cfg.Interval,
		reachableCap:         cfg.ReachableCap,
		contactConcurrency:   cfg.ContactConcurrency,
		self:                 self,
		reportedHostObserver: reportedHostObserver,
		seeds:                seeds,
		roster:               roster,
		greeter:              newPeerGreeter(exchange, cfg.NetworkName),
		observer:             observer,
	}
}
