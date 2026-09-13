// Package nodepeerhash keeps the peer hash the node answers to for the life of its
// data. The hash places the node on the DHT ring, so the stored postings belong
// to that position: the node adopts a hash on the first start over an empty data
// directory and refuses to run under a different one afterwards.
package nodepeerhash

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

var errInitialHashRejected = errors.New("configured peer hash differs from the stored peer hash")

type PeerHash struct {
	storage        *vault.Vault
	peerHashByPeer *peerHashByPeer
}

func Open(storage *vault.Vault) (*PeerHash, error) {
	peerHashByPeer, err := registerPeerHashByPeer(storage)
	if err != nil {
		return nil, err
	}

	return &PeerHash{storage: storage, peerHashByPeer: peerHashByPeer}, nil
}

func (p *PeerHash) Settle(
	ctx context.Context,
	initialHash yacymodel.Optional[yacymodel.Hash],
) (yacymodel.Hash, error) {
	stored, found, err := p.storedHash(ctx)
	if err != nil {
		return yacymodel.Hash{}, err
	}
	if found {
		if err := confirmHashUnchanged(stored, initialHash); err != nil {
			return yacymodel.Hash{}, err
		}

		return stored, nil
	}

	return p.adoptHash(ctx, initialHash)
}

func (p *PeerHash) storedHash(ctx context.Context) (yacymodel.Hash, bool, error) {
	var (
		stored yacymodel.Hash
		found  bool
	)

	if err := p.storage.View(ctx, func(tx *vault.Txn) error {
		hash, present, err := p.peerHashByPeer.Get(tx, selfPeer)
		stored, found = hash, present

		return err
	}); err != nil {
		return yacymodel.Hash{}, false, fmt.Errorf("read stored peer hash: %w", err)
	}

	return stored, found, nil
}

func confirmHashUnchanged(
	stored yacymodel.Hash,
	initialHash yacymodel.Optional[yacymodel.Hash],
) error {
	configured, present := initialHash.Get()
	if present && configured != stored {
		return fmt.Errorf("%w: %s is stored, %s is configured",
			errInitialHashRejected, stored, configured)
	}

	return nil
}

func (p *PeerHash) adoptHash(
	ctx context.Context,
	initialHash yacymodel.Optional[yacymodel.Hash],
) (yacymodel.Hash, error) {
	adopted, err := hashToAdopt(initialHash)
	if err != nil {
		return yacymodel.Hash{}, err
	}

	if err := p.storage.Update(ctx, func(tx *vault.Txn) error {
		return p.peerHashByPeer.Put(tx, selfPeer, adopted)
	}); err != nil {
		return yacymodel.Hash{}, fmt.Errorf("store peer hash: %w", err)
	}

	slog.InfoContext(ctx, "peer hash adopted for the life of this data directory",
		slog.String("peer", adopted.String()),
		slog.Bool("configured", initialHash.Present()),
	)

	return adopted, nil
}

func hashToAdopt(
	initialHash yacymodel.Optional[yacymodel.Hash],
) (yacymodel.Hash, error) {
	if configured, present := initialHash.Get(); present {
		return configured, nil
	}

	generated, err := yacymodel.NewHash()
	if err != nil {
		return yacymodel.Hash{}, fmt.Errorf("generate peer hash: %w", err)
	}

	return generated, nil
}
