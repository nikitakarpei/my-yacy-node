package peerhash

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

const peerHashBucket vault.Name = "peerhash_peers"

const ownPeer = "own"

type peerHashCollection = vault.Collection[string, yacymodel.Hash]

func registerPeerHashes(storage *vault.Vault) (*peerHashCollection, error) {
	peerHashes, err := vault.RegisterCollection(
		storage,
		peerHashBucket,
		peerKeyCodec{},
		peerHashValueCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register peer hashes: %w", err)
	}

	return peerHashes, nil
}

var peerKeyLayout = vaultkey.Single(vaultkey.Text)

type peerKeyCodec struct{}

func (peerKeyCodec) Encode(peer string) vaultkey.Key {
	return peerKeyLayout.Key(peer)
}

func (peerKeyCodec) Decode(storedKey []byte) (string, error) {
	peer, err := peerKeyLayout.Parts(storedKey)
	if err != nil {
		return "", fmt.Errorf("peer key: %w", err)
	}

	return peer, nil
}

type peerHashValueCodec struct{}

func (peerHashValueCodec) Encode(hash yacymodel.Hash) ([]byte, error) {
	return []byte(hash.String()), nil
}

func (peerHashValueCodec) Decode(payload []byte) (yacymodel.Hash, error) {
	hash, err := yacymodel.ParseHash(string(payload))
	if err != nil {
		return yacymodel.Hash{}, fmt.Errorf("stored peer hash: %w", err)
	}

	return hash, nil
}
