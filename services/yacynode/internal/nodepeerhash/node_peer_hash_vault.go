package nodepeerhash

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const peerHashBucket vault.Name = "nodepeerhash"

const selfPeer = "self"

type peerHashByPeer = vault.Collection[string, yacymodel.Hash]

func registerPeerHashByPeer(storage *vault.Vault) (*peerHashByPeer, error) {
	peerHashByPeer, err := vault.RegisterCollection(
		storage,
		peerHashBucket,
		peerKeyCodec{},
		peerHashValueCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register peer hash: %w", err)
	}

	return peerHashByPeer, nil
}

var peerKeyLayout = vault.SingleKey(vault.TextKeyPart)

type peerKeyCodec struct{}

func (peerKeyCodec) Encode(peer string) vault.Key {
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
