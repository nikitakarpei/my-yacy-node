// Package peerhash keeps the peer hash the node answers to for the life of its
// data. The hash is adopted once, on the first start over an empty data
// directory, and stays with that data: the postings, replica records and roster
// entries beside it are keyed on it, and the peers holding copies address the
// node by it. A hash the operator states is therefore taken only while nothing
// is stored yet; afterwards it must match, or the node refuses to run under an
// identity its data does not belong to.
package peerhash

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

var errInitialHashRejected = errors.New("stated peer hash differs from the stored peer hash")

func Settle(
	ctx context.Context,
	storage *vault.Vault,
	initialHash yacymodel.Optional[yacymodel.Hash],
) (yacymodel.Hash, error) {
	peerHashes, err := registerPeerHashes(storage)
	if err != nil {
		return yacymodel.Hash{}, err
	}

	stored, found, err := storedHash(ctx, storage, peerHashes)
	if err != nil {
		return yacymodel.Hash{}, err
	}
	if found {
		return confirmedHash(stored, initialHash)
	}

	return adoptHash(ctx, storage, peerHashes, initialHash)
}

func storedHash(
	ctx context.Context,
	storage *vault.Vault,
	peerHashes *peerHashCollection,
) (yacymodel.Hash, bool, error) {
	var (
		stored yacymodel.Hash
		found  bool
	)

	if err := storage.View(ctx, func(tx *vault.Txn) error {
		hash, present, err := peerHashes.Get(tx, ownPeer)
		stored, found = hash, present

		return err
	}); err != nil {
		return yacymodel.Hash{}, false, fmt.Errorf("read stored peer hash: %w", err)
	}

	return stored, found, nil
}

func confirmedHash(
	stored yacymodel.Hash,
	initialHash yacymodel.Optional[yacymodel.Hash],
) (yacymodel.Hash, error) {
	stated, present := initialHash.Get()
	if present && stated != stored {
		return yacymodel.Hash{}, fmt.Errorf("%w: %s is stored, %s was stated",
			errInitialHashRejected, stored, stated)
	}

	return stored, nil
}

func adoptHash(
	ctx context.Context,
	storage *vault.Vault,
	peerHashes *peerHashCollection,
	initialHash yacymodel.Optional[yacymodel.Hash],
) (yacymodel.Hash, error) {
	adopted, err := hashToAdopt(initialHash)
	if err != nil {
		return yacymodel.Hash{}, err
	}

	if err := storage.Update(ctx, func(tx *vault.Txn) error {
		return peerHashes.Put(tx, ownPeer, adopted)
	}); err != nil {
		return yacymodel.Hash{}, fmt.Errorf("store peer hash: %w", err)
	}

	slog.InfoContext(ctx, "peer hash adopted for the life of this data directory",
		slog.String("peer", adopted.String()),
		slog.Bool("stated", initialHash.Present()),
	)

	return adopted, nil
}

func hashToAdopt(
	initialHash yacymodel.Optional[yacymodel.Hash],
) (yacymodel.Hash, error) {
	if stated, present := initialHash.Get(); present {
		return stated, nil
	}

	generated, err := yacymodel.NewHash()
	if err != nil {
		return yacymodel.Hash{}, fmt.Errorf("generate peer hash: %w", err)
	}

	return generated, nil
}
