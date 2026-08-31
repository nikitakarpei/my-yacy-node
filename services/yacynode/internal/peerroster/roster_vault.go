package peerroster

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
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
	stored.Text(entry.primaryAddress.String())
	stored.Varint(int64(entry.port))

	return stored.Record(), nil
}

func (rosterEntryValueCodec) Decode(raw []byte) (rosterEntry, error) {
	stored := storedfields.ReaderOf(raw, errBadRosterEntry)
	entry := rosterEntry{
		lastReachable:  stored.Time("last reachable"),
		lastContacted:  stored.Time("last contacted"),
		primaryAddress: primaryAddressFrom(stored),
		port:           portFrom(stored),
	}
	if err := stored.Err(); err != nil {
		return rosterEntry{}, err
	}

	return entry, nil
}

func primaryAddressFrom(stored *storedfields.Reader) yacymodel.Host {
	address, err := yacymodel.ParseHost(stored.Text("primary address"))
	if err != nil {
		stored.Reject("primary address", err)
	}

	return address
}

func portFrom(stored *storedfields.Reader) yacymodel.Port {
	port, err := yacymodel.ParsePort(strconv.FormatInt(stored.Varint("port"), 10))
	if err != nil {
		stored.Reject("port", err)
	}

	return port
}
