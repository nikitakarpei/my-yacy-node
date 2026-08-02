package main

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmetastaleness"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlreferences"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type nodeStorage struct {
	vault           *vault.Vault
	now             func() time.Time
	urlDirectory    urlmeta.URLDirectory
	urlEvictor      urlmeta.URLEvictor
	urlReceiver     urlmeta.URLReceiver
	staleness       urlmetastaleness.StalenessRanking
	references      urlreferences.ReferenceProjection
	postings        rwipostings.PostingIndex
	postingReceiver rwiadmission.PostingReceiver
	postingPurger   rwipostings.PostingPurger
	escrow          *rwiescrow.HeldPostings
	offerSchedule   *postingofferschedule.Schedule
	replicas        *postingreplicas.Replicas
}

func openNodeStorage(
	vault *vault.Vault,
	now func() time.Time,
	escrowObserver rwiescrow.HoldObserver,
	offerObserver postingofferschedule.Observer,
) (
	nodeStorage,
	error,
) {
	staleness, err := urlmetastaleness.Open(vault)
	if err != nil {
		return nodeStorage{}, fmt.Errorf("url metadata staleness: %w", err)
	}

	references, err := urlreferences.Open(vault)
	if err != nil {
		return nodeStorage{}, fmt.Errorf("url references: %w", err)
	}

	offerSchedule, replicas, postingRecords, err := rwidistribution.Open(vault, now, offerObserver)
	if err != nil {
		return nodeStorage{}, fmt.Errorf("rwi distribution: %w", err)
	}

	postings, admitter, postingPurger, err := rwipostings.Open(
		vault,
		references,
		postingRecords,
	)
	if err != nil {
		return nodeStorage{}, fmt.Errorf("rwi storage: %w", err)
	}

	escrow, err := rwiescrow.Open(vault, admitter, escrowObserver, escrowQuotaFraction, now)
	if err != nil {
		return nodeStorage{}, fmt.Errorf("rwi escrow: %w", err)
	}

	urlDirectory, urlEvictor, urlReceiver, err := urlmeta.Open(vault, staleness, escrow)
	if err != nil {
		return nodeStorage{}, fmt.Errorf("urlmeta storage: %w", err)
	}

	return nodeStorage{
		vault:        vault,
		now:          now,
		urlDirectory: urlDirectory,
		urlEvictor:   urlEvictor,
		urlReceiver:  urlReceiver,
		staleness:    staleness,
		references:   references,
		postings:     postings,
		postingReceiver: rwiadmission.Open(
			vault,
			urlDirectory,
			admitter,
			escrow,
			rwiadmission.Config{BatchCap: receiveBatchCap, Pause: receiveBusyPause},
		),
		postingPurger: postingPurger,
		escrow:        escrow,
		offerSchedule: offerSchedule,
		replicas:      replicas,
	}, nil
}
