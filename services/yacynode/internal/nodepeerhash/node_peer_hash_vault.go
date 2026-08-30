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
	peerHashByPeer, err := storage.RegisterCollection(
		peerHashBucket,
		peerKeyCodec,
		peerHashValueCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register peer hash: %w", err)
	}

	return peerHashByPeer, nil
}

var peerKeyCodec = vault.SingleKey(vault.TextKeyPart).KeyCodec()

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
