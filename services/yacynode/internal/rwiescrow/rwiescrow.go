// Package rwiescrow holds an inbound RWI posting outside the index until the URL
// metadata it names arrives. An escrowed posting joins the index inside the
// transaction that stores that metadata, and is dropped when its hold outlives the
// hold period. The index therefore carries only postings this node can redistribute.
package rwiescrow

import (
	"context"
	"errors"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

var (
	_ urlmeta.URLMetadataObserver = (*PostingEscrow)(nil)
	_ PostingExpiry               = (*PostingEscrow)(nil)
)

var ErrEscrowFull = errors.New("escrow holds as many postings as it can")

type HoldObserver interface {
	ObserveHeld(postings int)
	ObserveReleased(postings int)
}

type ExpiryObserver interface {
	ObserveExpired(postings int)
	ObserveExpiryFailure()
}

type PostingExpiry interface {
	Expire(ctx context.Context, holdFor time.Duration, limit int) (int, error)
}

type ExpiryConfig struct {
	HoldFor  time.Duration
	Interval time.Duration
	Batch    int
}

func Open(
	v *vault.Vault,
	admitter rwipostings.PostingAdmitter,
	observer HoldObserver,
	capacity int,
	now func() time.Time,
) (*PostingEscrow, error) {
	escrowed, holds, err := registerPostingEscrow(v)
	if err != nil {
		return nil, err
	}

	return &PostingEscrow{
		vault:    v,
		escrowed: escrowed,
		holds:    holds,
		admitter: admitter,
		observer: observer,
		capacity: capacity,
		now:      now,
	}, nil
}
