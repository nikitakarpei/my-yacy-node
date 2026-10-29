package peertls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

var (
	certificateValidFrom  = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	certificateValidUntil = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	serialNumberLimit     = new(big.Int).Lsh(big.NewInt(1), 128)
)

func ServerConfigFor(key yacymodel.PeerKey, networkName string) (*tls.Config, error) {
	certificate, err := certificateFor(key, networkName)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS13,
	}, nil
}

func certificateFor(key yacymodel.PeerKey, networkName string) (tls.Certificate, error) {
	tlsKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("make the TLS key: %w", err)
	}
	tlsPublicKey, err := x509.MarshalPKIXPublicKey(&tlsKey.PublicKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("encode the TLS key: %w", err)
	}
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("make the serial number: %w", err)
	}
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: key.Hash().String()},
		NotBefore:    certificateValidFrom,
		NotAfter:     certificateValidUntil,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		URIs:         []*url.URL{endorsementURIOf(key.EndorsementOf(tlsPublicKey, networkName))},
	}
	certificate, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&tlsKey.PublicKey,
		tlsKey,
	)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("make the certificate: %w", err)
	}

	return tls.Certificate{Certificate: [][]byte{certificate}, PrivateKey: tlsKey}, nil
}
