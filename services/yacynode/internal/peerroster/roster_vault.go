package peerroster

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const peersBucket vault.Name = "peerroster"

func registerRoster(v *vault.Vault) (*vault.Collection[yacymodel.Hash, rosterEntry], error) {
	peers, err := vault.RegisterCollection(v, peersBucket, peerKeyCodec{}, rosterEntryValueCodec{})
	if err != nil {
		return nil, fmt.Errorf("register peer roster: %w", err)
	}

	return peers, nil
}

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

type rosterEntryValueCodec struct{}

func (rosterEntryValueCodec) Encode(entry rosterEntry) ([]byte, error) {
	contactTimes, err := binary.Append(nil, binary.BigEndian, []int64{
		entry.lastReachable.Unix(),
		int64(entry.lastReachable.Nanosecond()),
		entry.lastContacted.Unix(),
		int64(entry.lastContacted.Nanosecond()),
	})
	if err != nil {
		return nil, fmt.Errorf("encode roster entry contact times: %w", err)
	}

	return append(contactTimes, yacyproto.EncodeSeed(entry.seed)...), nil
}

func (rosterEntryValueCodec) Decode(raw []byte) (rosterEntry, error) {
	contactTimes := make([]int64, 4)
	contactTimesLength, err := binary.Decode(raw, binary.BigEndian, contactTimes)
	if err != nil {
		return rosterEntry{}, fmt.Errorf("roster entry contact times: %w", err)
	}
	reachableSeconds, reachableNanoseconds := contactTimes[0], contactTimes[1]
	contactedSeconds, contactedNanoseconds := contactTimes[2], contactTimes[3]

	seed, err := yacyproto.ParseSeed(context.Background(), string(raw[contactTimesLength:]))
	if err != nil {
		return rosterEntry{}, fmt.Errorf("decode roster entry: %w", err)
	}

	return rosterEntry{
		seed:          seed,
		lastReachable: time.Unix(reachableSeconds, reachableNanoseconds).UTC(),
		lastContacted: time.Unix(contactedSeconds, contactedNanoseconds).UTC(),
	}, nil
}
