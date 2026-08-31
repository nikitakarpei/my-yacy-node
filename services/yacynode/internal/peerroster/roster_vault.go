package peerroster

import (
	"context"
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const peersBucket vault.Name = "peerroster"

var errBadRosterEntry = errors.New("bad roster entry")

func registerRoster(v *vault.Vault) (*vault.Collection[yacymodel.Hash, rosterEntry], error) {
	peers, err := v.RegisterCollection(peersBucket, peerKeyLayout, rosterEntryValueCodec{})
	if err != nil {
		return nil, fmt.Errorf("register peer roster: %w", err)
	}

	return peers, nil
}

var peerKeyLayout = vault.SingleKey(hashkeypart.Hash).KeyLayout()

type rosterEntryValueCodec struct{}

func (rosterEntryValueCodec) Encode(entry rosterEntry) ([]byte, error) {
	var stored storedfields.Writer
	stored.Time(entry.lastReachable)
	stored.Time(entry.lastContacted)
	stored.Text(yacyproto.EncodeSeed(entry.seed))

	return stored.Record(), nil
}

func (rosterEntryValueCodec) Decode(raw []byte) (rosterEntry, error) {
	stored := storedfields.ReaderOf(raw, errBadRosterEntry)
	entry := rosterEntry{
		lastReachable: stored.Time("last reachable"),
		lastContacted: stored.Time("last contacted"),
	}
	encodedSeed := stored.Text("seed")
	if err := stored.Err(); err != nil {
		return rosterEntry{}, err
	}

	seed, err := yacyproto.ParseSeed(context.Background(), encodedSeed)
	if err != nil {
		return rosterEntry{}, fmt.Errorf("%w: %w", errBadRosterEntry, err)
	}
	entry.seed = seed

	return entry, nil
}
