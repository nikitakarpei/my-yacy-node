// Package brokeraccesstest starts a NATS broker with TLS for the tests of the app, makes device
// keys that sign in software, and writes pair links to the broker.
package brokeraccesstest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

const (
	readyWait = 10 * time.Second
	PeerUser  = "peer"
)

type Broker struct {
	Server  *natsserver.Server
	KeyHash companionprotocol.BrokerKeyHash
}

func Start(t *testing.T, authentication natsserver.Authentication) Broker {
	t.Helper()
	certificate, keyHash := brokerCertificate(t)
	server, err := natsserver.NewServer(&natsserver.Options{
		Host:      "127.0.0.1",
		Port:      natsserver.RANDOM_PORT,
		JetStream: true,
		StoreDir:  t.TempDir(),
		TLS:       true,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{certificate},
			ClientAuth:   tls.RequestClientCert,
			MinVersion:   tls.VersionTLS13,
		},
		CustomClientAuthentication: authentication,
	})
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}
	go server.Start()
	if !server.ReadyForConnections(readyWait) {
		t.Fatal("broker not ready")
	}
	t.Cleanup(server.Shutdown)
	return Broker{Server: server, KeyHash: keyHash}
}

func brokerCertificate(t *testing.T) (tls.Certificate, companionprotocol.BrokerKeyHash) {
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
		companionprotocol.BrokerKeyHashFrom(leaf.RawSubjectPublicKeyInfo)
}

func (b Broker) Address() string {
	return b.Server.Addr().String()
}

func (b Broker) LinkWith(secret companionprotocol.Secret) string {
	return companionprotocol.PairLink{
		Addresses:     []string{b.Address()},
		BrokerKeyHash: b.KeyHash,
		Secret:        secret,
	}.String()
}

func (b Broker) Connect(t *testing.T) *nats.Conn {
	t.Helper()
	connection, err := nats.Connect("tls://"+b.Address(), nats.Secure(&tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // the test connects to its own broker.
	}), nats.UserInfo(PeerUser, ""))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(connection.Close)
	return connection
}

type DeviceKey struct {
	key *ecdsa.PrivateKey
}

func RandomDeviceKey(t *testing.T) DeviceKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return DeviceKey{key: key}
}

func (k DeviceKey) PublicKeyInfo() []byte {
	info, _ := x509.MarshalPKIXPublicKey(&k.key.PublicKey)
	return info
}

func (k DeviceKey) SignDigest(digest []byte) ([]byte, error) {
	return ecdsa.SignASN1(rand.Reader, k.key, digest) //nolint:wrapcheck // test key.
}

func (k DeviceKey) ID() companionprotocol.DeviceID {
	return companionprotocol.DeviceIDFrom(k.PublicKeyInfo())
}
