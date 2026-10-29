package peertls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

var (
	errNoCertificate = errors.New("no certificate")
	errNoEndorsement = errors.New("no TLS key endorsement")
	errOtherIdentity = errors.New("certificate of another identity")
	errNotEndorsed   = errors.New("TLS key not endorsed")
)

type Transports struct {
	proxyURL    *url.URL
	networkName string
}

func NewTransports(proxyURL *url.URL, networkName string) Transports {
	return Transports{proxyURL: proxyURL, networkName: networkName}
}

func (t Transports) TransportFor(identity yacymodel.PeerIdentity) http.RoundTripper {
	return &http.Transport{
		DialTLSContext: func(ctx context.Context, _, address string) (net.Conn, error) {
			return t.verifiedConnectionTo(ctx, address, identity)
		},
		DisableKeepAlives: true,
	}
}

func (t Transports) verifiedConnectionTo(
	ctx context.Context,
	address string,
	identity yacymodel.PeerIdentity,
) (net.Conn, error) {
	tunnel, err := tunnelTo(ctx, t.proxyURL, address)
	if err != nil {
		return nil, err
	}
	connection := tls.Client(tunnel, clientConfigFor(identity, t.networkName))
	if err := connection.HandshakeContext(ctx); err != nil {
		return nil, errors.Join(
			fmt.Errorf("TLS handshake with %s: %w", address, err),
			tunnel.Close(),
		)
	}

	return connection, nil
}

func clientConfigFor(identity yacymodel.PeerIdentity, networkName string) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		//nolint:gosec // G402: VerifyConnection checks the certificate against the peer key.
		InsecureSkipVerify: true,
		VerifyConnection: func(state tls.ConnectionState) error {
			return verifyIdentity(state, identity, networkName)
		},
	}
}

func verifyIdentity(
	state tls.ConnectionState,
	identity yacymodel.PeerIdentity,
	networkName string,
) error {
	if len(state.PeerCertificates) == 0 {
		return errNoCertificate
	}
	certificate := state.PeerCertificates[0]
	endorsement, err := endorsementOf(certificate)
	if err != nil {
		return err
	}
	if endorsement.Identity != identity {
		return fmt.Errorf("%w: peer %s, want peer %s",
			errOtherIdentity, endorsement.Identity.Hash(), identity.Hash())
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
