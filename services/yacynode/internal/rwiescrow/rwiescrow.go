// Package rwiescrow holds inbound RWI postings outside the index while they
// wait for URL metadata of their page version, gives back the postings waiting
// for a URL, and drops a posting whose hold outlives the hold period. The index
// therefore carries only postings this node can redistribute.
package rwiescrow

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

var _ PostingExpiry = (*PostingEscrow)(nil)

type HoldObserver interface {
	ObserveHeld(tx *vault.Txn, postings int)
	ObserveReleased(tx *vault.Txn, postings int)
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
	ctx context.Context,
	v *vault.Vault,
	observer HoldObserver,
	capacity int,
	now func() time.Time,
) (*PostingEscrow, error) {
	escrowed, holds, err := registerPostingEscrow(ctx, v)
	if err != nil {
		return nil, err
	}

	return &PostingEscrow{
		vault:    v,
		escrowed: escrowed,
		holds:    holds,
		observer: observer,
		capacity: capacity,
		now:      now,
	}, nil
}
