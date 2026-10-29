package peerroster

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
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
	stored.Text(entry.addresses.Hosts[0].String())
	stored.Varint(int64(entry.addresses.Port))
	additionalHosts := entry.addresses.Hosts[1:]
	stored.Count(len(additionalHosts))
	for _, host := range additionalHosts {
		stored.Text(host.String())
	}
	stored.Varint(int64(entry.addresses.SecurePort.OrElse(0)))
	answeredBaseURL, answered := entry.lastAnsweredBaseURL.Get()
	stored.Text(string(answeredBaseURL.Scheme))
	if answered {
		stored.Text(answeredBaseURL.Host.String())
		stored.Varint(int64(answeredBaseURL.Port))
	}
	binding, bound := entry.binding.Get()
	stored.Presence(bound)
	if bound {
		stored.Text(binding.Identity.String())
		stored.Time(binding.IssuedAt)
	}

	return stored.Record(), nil
}

func (rosterEntryValueCodec) Decode(raw []byte) (rosterEntry, error) {
	stored := storedfields.ReaderOf(raw, errBadRosterEntry)
	entry := rosterEntry{
		lastReachable: stored.Time("last reachable"),
		lastContacted: stored.Time("last contacted"),
		addresses:     addressesFrom(stored),
	}
	if stored.BytesLeft() > 0 {
		entry.lastAnsweredBaseURL = lastAnsweredBaseURLFrom(stored)
	}
	if stored.BytesLeft() > 0 {
		entry.binding = storedBindingFrom(stored)
	}
	if err := stored.Err(); err != nil {
		return rosterEntry{}, err
	}

	return entry, nil
}

func addressesFrom(stored *storedfields.Reader) yacymodel.PeerAddresses {
	addresses := yacymodel.PeerAddresses{
		Hosts: []yacymodel.Host{hostFrom(stored, "primary host")},
		Port:  portFrom(stored, "port"),
	}
	if stored.BytesLeft() == 0 {
		return addresses
	}
	for range stored.Count("additional hosts") {
		addresses.Hosts = append(addresses.Hosts, hostFrom(stored, "additional host"))
	}
	addresses.SecurePort = securePortFrom(stored)

	return addresses
}

func lastAnsweredBaseURLFrom(stored *storedfields.Reader) yacymodel.Optional[peerwire.BaseURL] {
	const field = "last answered base URL scheme"
	text := stored.Text(field)
	if text == "" {
		return yacymodel.None[peerwire.BaseURL]()
	}
	scheme, err := peerwire.ParseScheme(text)
	if err != nil {
		stored.Reject(field, err)
	}

	return yacymodel.Some(peerwire.BaseURL{
		Scheme: scheme,
		Host:   hostFrom(stored, "last answered base URL host"),
		Port:   portFrom(stored, "last answered base URL port"),
	})
}

func storedBindingFrom(
	stored *storedfields.Reader,
) yacymodel.Optional[yacymodel.VerifiedAddressSignature] {
	if !stored.Presence("bound") {
		return yacymodel.None[yacymodel.VerifiedAddressSignature]()
	}
	const field = "bound identity"
	identity, err := yacymodel.ParsePeerIdentity(stored.Text(field))
	if err != nil {
		stored.Reject(field, err)
	}

	return yacymodel.Some(
		yacymodel.VerifiedAddressSignature{
			Identity: identity,
			IssuedAt: stored.Time("bound issue time"),
		},
	)
}

func hostFrom(stored *storedfields.Reader, field string) yacymodel.Host {
	host, err := yacymodel.ParseHost(stored.Text(field))
	if err != nil {
		stored.Reject(field, err)
	}

	return host
}

func portFrom(stored *storedfields.Reader, field string) yacymodel.Port {
	return portFromNumber(stored, field, stored.Varint(field))
}

func securePortFrom(stored *storedfields.Reader) yacymodel.Optional[yacymodel.Port] {
	const field = "secure port"
	number := stored.Varint(field)
	if number == 0 {
		return yacymodel.None[yacymodel.Port]()
	}

	return yacymodel.Some(portFromNumber(stored, field, number))
}

func portFromNumber(stored *storedfields.Reader, field string, number int64) yacymodel.Port {
	port, err := yacymodel.ParsePort(strconv.FormatInt(number, 10))
	if err != nil {
		stored.Reject(field, err)
	}

	return port
}
