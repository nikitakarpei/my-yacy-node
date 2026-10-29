package peertls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

var (
	errNoCertificate = errors.New("no certificate")
	errNoEndorsement = errors.New("no TLS key endorsement")
	errOtherPeer     = errors.New("certificate of another peer")
	errNotEndorsed   = errors.New("TLS key not endorsed")
)

type Transports struct {
	proxy       func(*http.Request) (*url.URL, error)
	networkName string
}

func NewTransports(proxy func(*http.Request) (*url.URL, error), networkName string) Transports {
	return Transports{proxy: proxy, networkName: networkName}
}

func (t Transports) TransportFor(peer yacymodel.Hash) http.RoundTripper {
	return &http.Transport{
		Proxy:             t.proxy,
		TLSClientConfig:   clientConfigFor(peer, t.networkName),
		DisableKeepAlives: true,
	}
}

func clientConfigFor(peer yacymodel.Hash, networkName string) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		//nolint:gosec // G402: VerifyConnection checks the certificate against the peer key.
		InsecureSkipVerify: true,
		VerifyConnection: func(state tls.ConnectionState) error {
			return checkedPeer(state, peer, networkName)
		},
	}
}

func checkedPeer(state tls.ConnectionState, peer yacymodel.Hash, networkName string) error {
	if len(state.PeerCertificates) == 0 {
		return errNoCertificate
	}
	certificate := state.PeerCertificates[0]
	endorsement, err := endorsementOf(certificate)
	if err != nil {
		return err
	}
	if endorsement.Identity.Hash() != peer {
		return fmt.Errorf("%w: %s, want %s", errOtherPeer, endorsement.Identity.Hash(), peer)
	}
	if !endorsement.Endorses(certificate.RawSubjectPublicKeyInfo, networkName) {
		return errNotEndorsed
	}

	return nil
}

func endorsementOf(certificate *x509.Certificate) (yacymodel.TLSKeyEndorsement, error) {
	for _, uri := range certificate.URIs {
		if uri.Scheme == endorsementScheme {
			return endorsementFrom(uri.Opaque)
		}
	}

	return yacymodel.TLSKeyEndorsement{}, errNoEndorsement
}

func endorsementFrom(text string) (yacymodel.TLSKeyEndorsement, error) {
	cut := strings.LastIndex(text, endorsementPartSeparator)
	if cut < 0 {
		return yacymodel.TLSKeyEndorsement{}, errNoEndorsement
	}
	identity, err := yacymodel.ParsePeerIdentity(text[:cut])
	if err != nil {
		return yacymodel.TLSKeyEndorsement{}, fmt.Errorf("%w: %w", errNoEndorsement, err)
	}
	signature, err := yacymodel.ParseSignature(text[cut+1:])
	if err != nil {
		return yacymodel.TLSKeyEndorsement{}, fmt.Errorf("%w: %w", errNoEndorsement, err)
	}

	return yacymodel.TLSKeyEndorsement{Identity: identity, Signature: signature}, nil
}
