// Package nodepeerhash keeps the peer hash the node answers to for the life of its
// data. The hash places the node on the DHT ring, so the stored postings belong
// to that position: the node stores the hash of its peer key on the first start
// over an empty data directory and refuses to run under another key afterwards.
package nodepeerhash

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

var errKeyHashRejected = errors.New("hash of the peer key differs from the stored peer hash")

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

func (p *PeerHash) Settle(ctx context.Context, keyHash yacymodel.Hash) error {
	stored, found, err := p.storedHash(ctx)
	if err != nil {
		return err
	}
	if found {
		return confirmHashUnchanged(stored, keyHash)
	}

	return p.storeHash(ctx, keyHash)
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

func confirmHashUnchanged(stored, keyHash yacymodel.Hash) error {
	if keyHash != stored {
		return fmt.Errorf("%w: %s is stored, the key derives %s",
			errKeyHashRejected, stored, keyHash)
	}

	return nil
}

func (p *PeerHash) storeHash(ctx context.Context, keyHash yacymodel.Hash) error {
	if err := p.storage.Update(ctx, func(tx *vault.Txn) error {
		_, err := p.peerHashByPeer.Put(tx, selfPeer, keyHash)

		return err
	}); err != nil {
		return fmt.Errorf("store peer hash: %w", err)
	}

	slog.InfoContext(ctx, "peer hash stored for the life of this data directory",
		slog.String("peer", keyHash.String()),
	)

	return nil
}
