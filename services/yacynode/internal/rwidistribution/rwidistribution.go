// Package rwidistribution offers stored RWI postings to the peers the DHT
// makes responsible for them and keeps offering until enough of them accept.
// It observes posting arrivals and departures to build its own due-ordered
// work queue, so it never scans the posting store to find what is owed.
package rwidistribution

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/distributioncycle"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingcourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/replicarecipients"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/replicashortfall"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/urlmetadatacourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type Config struct {
	NetworkName          string
	Self                 yacymodel.Hash
	Redundancy           int
	Partitions           yacymodel.DHTRingPartitions
	PostingsPerCycle     int
	CycleInterval        time.Duration
	RefreshInterval      time.Duration
	RetryInterval        time.Duration
	RecipientCooldown    time.Duration
	MinReachablePeers    int
	URLMetadataBatchSize int
}

type Runner interface {
	Run(ctx context.Context)
}

type PostingOfferCycleObserver = distributioncycle.Observer

type Distribution struct {
	schedule *postingschedule.Schedule
	replicas *postingreplicas.Replicas
	now      func() time.Time
}

func Open(v *vault.Vault, now func() time.Time) (*Distribution, error) {
	schedule, err := postingschedule.Open(v, now)
	if err != nil {
		return nil, fmt.Errorf("open offer schedule: %w", err)
	}

	replicas, err := postingreplicas.Open(v, schedule)
	if err != nil {
		return nil, fmt.Errorf("open replica ledger: %w", err)
	}

	return &Distribution{schedule: schedule, replicas: replicas, now: now}, nil
}

func (d *Distribution) PostingStored(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	return d.schedule.PostingStored(tx, word, url)
}

func (d *Distribution) PostingPurged(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	if err := d.schedule.PostingPurged(tx, word, url); err != nil {
		return err
	}

	return d.replicas.PostingPurged(tx, word, url)
}

//nolint:revive // argument-limit: six explicit, independently-meaningful collaborators
func (d *Distribution) Cycle(
	client *http.Client,
	postings rwipostings.PostingIndex,
	roster peerroster.Roster,
	urls urlmeta.URLDirectory,
	observer PostingOfferCycleObserver,
	cfg Config,
) Runner {
	exchange := peerwire.NewMessageExchange(client)

	recipients := replicarecipients.New(cfg.RecipientCooldown, d.now)

	shortfall := replicashortfall.New(
		d.schedule, d.replicas, postings, roster, recipients, cfg.Partitions, cfg.Redundancy,
	)
	delivery := distributioncycle.NewOfferDelivery(
		postingcourier.New(exchange, cfg.NetworkName, cfg.Self),
		urlmetadatacourier.NewBounded(
			urlmetadatacourier.NewHTTP(
				exchange,
				cfg.NetworkName,
				cfg.Self,
			),
			cfg.URLMetadataBatchSize,
		),
		urls,
		observer,
	)
	return distributioncycle.New(
		shortfall,
		delivery,
		d.replicas,
		recipients,
		distributioncycle.Cadence{Refresh: cfg.RefreshInterval, Backoff: cfg.RetryInterval},
		d.schedule,
		roster,
		observer,
		d.now,
		cfg.PostingsPerCycle,
		cfg.CycleInterval,
		cfg.MinReachablePeers,
	)
}

var _ rwipostings.PostingObserver = (*Distribution)(nil)
