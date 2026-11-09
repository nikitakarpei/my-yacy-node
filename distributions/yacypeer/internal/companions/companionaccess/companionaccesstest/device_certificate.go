// Package companionaccesstest makes the certificates of companion devices for the tests of the
// peer.
package companionaccesstest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func DeviceCertificate(t *testing.T) (tls.Certificate, companionprotocol.DeviceID) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	certificateDER, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&key.PublicKey,
		key,
	)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(certificateDER)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{certificateDER}, PrivateKey: key},
		companionprotocol.DeviceIDFrom(leaf.RawSubjectPublicKeyInfo)
}
