// Package companionprotocol holds the wire protocol between the YaCy companion app and a peer:
// the pair link that the peer shows as a QR code, the credentials, request and reply of pairing,
// and the subjects that a paired device uses on the broker of the peer. The link names each
// address of the broker that a device can try, most likely first.
package companionprotocol

import (
	"errors"
	"net/url"
	"slices"
)

const (
	linkScheme = "yacypeer-companion"

	pairHost               = "pair"
	parameterAddress       = "address"
	parameterBrokerKeyHash = "brokerKeyHash"
	parameterSecret        = "secret"
)

var (
	errNotACompanionLink = errors.New("not a link for the YaCy companion app")
	ErrUnknownLinkKind   = errors.New("a companion link of a kind that this app does not know")
	ErrUnreadableLink    = errors.New("a pair link that this app cannot read")
)

type PairLink struct {
	Addresses     []string
	BrokerKeyHash BrokerKeyHash
	Secret        Secret
}

func (l PairLink) String() string {
	return (&url.URL{
		Scheme: linkScheme,
		Host:   pairHost,
		RawQuery: url.Values{
			parameterAddress:       l.Addresses,
			parameterBrokerKeyHash: {string(l.BrokerKeyHash)},
			parameterSecret:        {string(l.Secret)},
		}.Encode(),
	}).String()
}

func PairLinkFrom(text string) (PairLink, error) {
	linkURL, err := url.Parse(text)
	if err != nil || linkURL.Scheme != linkScheme {
		return PairLink{}, errNotACompanionLink
	}
	if linkURL.Host != pairHost {
		return PairLink{}, ErrUnknownLinkKind
	}
	parameters := linkURL.Query()
	link := PairLink{
		Addresses:     slices.DeleteFunc(parameters[parameterAddress], isEmpty),
		BrokerKeyHash: BrokerKeyHash(parameters.Get(parameterBrokerKeyHash)),
		Secret:        Secret(parameters.Get(parameterSecret)),
	}
	if len(link.Addresses) == 0 || link.BrokerKeyHash == "" || link.Secret == "" {
		return PairLink{}, ErrUnreadableLink
	}
	return link, nil
}

func isEmpty(address string) bool {
	return address == ""
}
