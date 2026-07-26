package main

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmetastaleness"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlreferences"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type nodeStorage struct {
	urlDirectory    urlmeta.URLDirectory
	urlEvictor      urlmeta.URLEvictor
	urlReceiver     urlmeta.URLReceiver
	staleness       urlmetastaleness.StalenessRanking
	references      urlreferences.ReferenceProjection
	postings        rwipostings.PostingIndex
	postingReceiver rwipostings.PostingReceiver
	postingPurger   rwipostings.PostingPurger
	distribution    *rwidistribution.Distribution
}

func openNodeStorage(vault *vault.Vault) (nodeStorage, error) {
	staleness, err := urlmetastaleness.Open(vault)
	if err != nil {
		return nodeStorage{}, fmt.Errorf("url metadata staleness: %w", err)
	}

	urlDirectory, urlEvictor, urlReceiver, err := urlmeta.Open(vault, staleness)
	if err != nil {
		return nodeStorage{}, fmt.Errorf("urlmeta storage: %w", err)
	}

	references, err := urlreferences.Open(vault)
	if err != nil {
		return nodeStorage{}, fmt.Errorf("url references: %w", err)
	}

	distribution, err := rwidistribution.Open(vault, time.Now)
	if err != nil {
		return nodeStorage{}, fmt.Errorf("rwi distribution: %w", err)
	}

	postings, postingReceiver, postingPurger, err := rwipostings.Open(
		vault,
		urlDirectory,
		rwipostings.Config{BatchCap: receiveBatchCap, PauseSeconds: receiveBusyPauseSecs},
		references,
		distribution,
	)
	if err != nil {
		return nodeStorage{}, fmt.Errorf("rwi storage: %w", err)
	}

	return nodeStorage{
		urlDirectory:    urlDirectory,
		urlEvictor:      urlEvictor,
		urlReceiver:     urlReceiver,
		staleness:       staleness,
		references:      references,
		postings:        postings,
		postingReceiver: postingReceiver,
		postingPurger:   postingPurger,
		distribution:    distribution,
	}, nil
}
