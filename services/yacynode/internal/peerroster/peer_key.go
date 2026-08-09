package peerroster

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var peerKeyLayout = vaultkey.Single(vaultkey.Text)

type peerKeyCodec struct{}

func (peerKeyCodec) Encode(peer yacymodel.Hash) vaultkey.Key {
	return peerKeyLayout.Key(peer.String())
}

func (peerKeyCodec) Decode(key vaultkey.Key) (yacymodel.Hash, error) {
	peer, err := peerKeyLayout.Parts(key)
	if err != nil {
		return yacymodel.Hash{}, fmt.Errorf("peer roster key: %w", err)
	}

	return yacymodel.ParseHash(peer)
}
