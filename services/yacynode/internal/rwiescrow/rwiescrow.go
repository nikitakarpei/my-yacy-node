// Package rwiescrow holds an inbound RWI posting outside the index until the URL
// metadata it names arrives. A held posting joins the index inside the transaction
// that stores that metadata, and is dropped when it waits longer than the hold
// period. The index therefore carries only postings this node can redistribute.
package rwiescrow

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

var (
	_ urlmeta.URLMetadataObserver = (*HeldPostings)(nil)
	_ PostingExpiry               = (*HeldPostings)(nil)
)

type HoldObserver interface {
	ObserveHeld(postings int)
	ObserveReleased(postings int)
	ObserveRefused(postings int)
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
) (*HeldPostings, error) {
	held, expiry, err := registerHeldPostings(v)
	if err != nil {
		return nil, err
	}

	return &HeldPostings{
		vault:    v,
		held:     held,
		expiry:   expiry,
		admitter: admitter,
		observer: observer,
		capacity: capacity,
		now:      now,
	}, nil
}
