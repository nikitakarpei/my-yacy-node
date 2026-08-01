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
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type Config struct {
	NetworkName       string
	Self              yacymodel.Hash
	Redundancy        int
	Partitions        yacymodel.DHTRingPartitions
	PostingsPerCycle  int
	CycleInterval     time.Duration
	RefreshInterval   time.Duration
	RetryInterval     time.Duration
	MinReachablePeers int
}

type Runner interface {
	Run(ctx context.Context)
}

type Distribution struct {
	schedule *postingOfferSchedule
	ledger   *replicaLedger
	now      func() time.Time
}

func Open(v *vault.Vault, now func() time.Time) (*Distribution, error) {
	schedule, err := openPostingOfferSchedule(v, now)
	if err != nil {
		return nil, fmt.Errorf("open offer schedule: %w", err)
	}

	ledger, err := openReplicaLedger(v, schedule)
	if err != nil {
		return nil, fmt.Errorf("open replica ledger: %w", err)
	}

	return &Distribution{schedule: schedule, ledger: ledger, now: now}, nil
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

	return d.ledger.PostingPurged(tx, word, url)
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
	exchange := peerMessageExchange{client: client}

	return &postingOfferCycle{
		reader: &postingReplicationReader{
			schedule:   d.schedule,
			ledger:     d.ledger,
			postings:   postings,
			roster:     roster,
			partitions: cfg.Partitions,
			redundancy: cfg.Redundancy,
		},
		postingCourier: httpPostingCourier{
			exchange:    exchange,
			networkName: cfg.NetworkName,
			self:        cfg.Self,
		},
		urlMetadataCourier: httpURLMetadataCourier{
			exchange:    exchange,
			networkName: cfg.NetworkName,
			self:        cfg.Self,
		},
		urls:     urls,
		schedule: d.schedule,
		ledger:   d.ledger,
		roster:   roster,
		observer: observer,
		cadence: postingOfferCadence{
			refresh: cfg.RefreshInterval,
			retry:   cfg.RetryInterval,
		},
		now:               d.now,
		postingsPerCycle:  cfg.PostingsPerCycle,
		cycleInterval:     cfg.CycleInterval,
		minReachablePeers: cfg.MinReachablePeers,
	}
}

var _ rwipostings.PostingObserver = (*Distribution)(nil)
