// Package rwiescrow holds an inbound RWI posting outside the index until URL
// metadata of its page version or newer arrives. The posting is then checked
// against the page version indexed for its URL, inside the transaction that
// stores that metadata, and is dropped when its hold outlives the hold period.
// The index therefore carries only postings this node can redistribute.
package rwiescrow

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlpageversion"
)

var (
	_ urlmeta.URLMetadataObserver = (*PostingEscrow)(nil)
	_ PostingExpiry               = (*PostingEscrow)(nil)
)

type PageVersions interface {
	Admit(
		tx *vault.Txn,
		origin rwiorigin.Origin,
		posting yacymodel.RWIPosting,
		pageVersion yacymodel.MicroDate,
	) (urlpageversion.Verdict, error)
}

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

//nolint:revive // argument-limit: independently-meaningful collaborators, not configuration to bundle further
func Open(
	ctx context.Context,
	v *vault.Vault,
	versions PageVersions,
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
		versions: versions,
		observer: observer,
		capacity: capacity,
		now:      now,
	}, nil
}
