package peertls_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peertls"
)

const networkName = "freeworld"

func TestTransportForAPeerReachesThePeerThatHoldsItsKey(t *testing.T) {
	server := peerServer(t, serverConfigOf(t, peerkeytest.Key(0)))

	if err := callOf(t, server, peerkeytest.Key(0).Hash(), networkName); err != nil {
		t.Fatalf("call = %v, want an answer", err)
	}
}

func TestTransportForAPeerRefusesAServerThatCannotProveItsKey(t *testing.T) {
	for name, refused := range map[string]struct {
		config  *tls.Config
		peer    yacymodel.Hash
		network string
	}{
		"another peer": {
			config:  serverConfigOf(t, peerkeytest.Key(1)),
			peer:    peerkeytest.Key(0).Hash(),
			network: networkName,
		},
		"another network": {
			config:  serverConfigOf(t, peerkeytest.Key(0)),
			peer:    peerkeytest.Key(0).Hash(),
			network: "othernet",
		},
		"no endorsement": {
			config:  configWithoutEndorsement(t),
			peer:    peerkeytest.Key(0).Hash(),
			network: networkName,
		},
		"endorsement of another TLS key": {
			config:  configWithCopiedEndorsement(t, serverConfigOf(t, peerkeytest.Key(0))),
			peer:    peerkeytest.Key(0).Hash(),
			network: networkName,
		},
	} {
		t.Run(name, func(t *testing.T) {
			server := peerServer(t, refused.config)

			if err := callOf(t, server, refused.peer, refused.network); err == nil {
				t.Fatal("call answered, want the server refused")
			}
		})
	}
}

func TestServerConfigMakesANewTLSKeyAtEachStart(t *testing.T) {
	first := leafOf(t, serverConfigOf(t, peerkeytest.Key(0)))
	second := leafOf(t, serverConfigOf(t, peerkeytest.Key(0)))

	if string(first.RawSubjectPublicKeyInfo) == string(second.RawSubjectPublicKeyInfo) {
		t.Error("two starts serve one TLS key, want a new key at each start")
	}
}

func serverConfigOf(t *testing.T, key yacymodel.PeerKey) *tls.Config {
	t.Helper()

	config, err := peertls.ServerConfigFor(key, networkName)
	if err != nil {
		t.Fatalf("server config: %v", err)
	}

	return config
}

func peerServer(t *testing.T, config *tls.Config) *httptest.Server {
	t.Helper()

	server := httptest.NewUnstartedServer(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	)
	server.TLS = config
	server.StartTLS()
	t.Cleanup(server.Close)

	return server
}

func callOf(t *testing.T, server *httptest.Server, peer yacymodel.Hash, network string) error {
	t.Helper()

	client := &http.Client{Transport: peertls.NewTransports(nil, network).TransportFor(peer)}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("call the peer: %w", err)
	}
	if err := response.Body.Close(); err != nil {
		return fmt.Errorf("close the answer: %w", err)
	}

	return nil
}

func leafOf(t *testing.T, config *tls.Config) *x509.Certificate {
	t.Helper()

	leaf, err := x509.ParseCertificate(config.Certificates[0].Certificate[0])
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}

	return leaf
}

func configWithoutEndorsement(t *testing.T) *tls.Config {
	t.Helper()

	return configOfSelfSignedCertificate(t, &x509.Certificate{SerialNumber: big.NewInt(1)})
}

func configWithCopiedEndorsement(t *testing.T, endorsed *tls.Config) *tls.Config {
	t.Helper()

	return configOfSelfSignedCertificate(t, &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "copied endorsement"},
		URIs:         leafOf(t, endorsed).URIs,
	})
}

func configOfSelfSignedCertificate(t *testing.T, template *x509.Certificate) *tls.Config {
	t.Helper()

	tlsKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("TLS key: %v", err)
	}
	template.NotAfter = template.NotBefore.AddDate(9000, 0, 0)
	certificate, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&tlsKey.PublicKey,
		tlsKey,
	)
	if err != nil {
		t.Fatalf("certificate: %v", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{certificate}, PrivateKey: tlsKey}},
		MinVersion:   tls.VersionTLS13,
	}
}
