// Package companionprotocol holds the wire protocol between the YaCy companion app and a peer:
// the pairing link that the peer shows as a QR code, the pairing request and its reply, and
// the subjects that a paired device uses on the broker of the peer. The link names each address
// of the broker that a device can try, most likely first.
package companionprotocol

import (
	"errors"
	"net/url"
	"slices"
)

const (
	Scheme = "yacypeer-companion"

	pairHost               = "pair"
	parameterAddress       = "address"
	parameterBrokerKeyHash = "brokerKeyHash"
	parameterSecret        = "secret"
)

var (
	ErrNotACompanionLink = errors.New("not a link for the YaCy companion app")
	ErrUnreadableLink    = errors.New("a companion link that this app cannot read")
)

type PairLink struct {
	Addresses     []string
	BrokerKeyHash string
	Secret        string
}

func (l PairLink) String() string {
	return (&url.URL{
		Scheme: Scheme,
		Host:   pairHost,
		RawQuery: url.Values{
			parameterAddress:       l.Addresses,
			parameterBrokerKeyHash: {l.BrokerKeyHash},
			parameterSecret:        {l.Secret},
		}.Encode(),
	}).String()
}

func PairLinkFrom(text string) (PairLink, error) {
	link, err := url.Parse(text)
	if err != nil || link.Scheme != Scheme {
		return PairLink{}, ErrNotACompanionLink
	}
	parameters := link.Query()
	pairLink := PairLink{
		Addresses:     slices.DeleteFunc(parameters[parameterAddress], isEmpty),
		BrokerKeyHash: parameters.Get(parameterBrokerKeyHash),
		Secret:        parameters.Get(parameterSecret),
	}
	if link.Host != pairHost || len(pairLink.Addresses) == 0 || pairLink.BrokerKeyHash == "" ||
		pairLink.Secret == "" {
		return PairLink{}, ErrUnreadableLink
	}
	return pairLink, nil
}

func isEmpty(address string) bool {
	return address == ""
}
