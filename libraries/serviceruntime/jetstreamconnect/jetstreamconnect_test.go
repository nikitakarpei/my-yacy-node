package jetstreamconnect_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
)

const callbackWait = 5 * time.Second

func TestOpenConnectsAndCloses(t *testing.T) {
	url := natstestserver.Start(t)
	js, connection, err := jetstreamconnect.Open(
		url, (&net.Dialer{}).DialContext,
		jetstreamconnect.UnobservedConnection{}, jetstreamconnect.BrokerAccess{},
	)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if js == nil {
		t.Fatal("Open returned nil jetstream")
	}
	connection.Close()
}

func TestOpenConnectsThroughTheDialer(t *testing.T) {
	broker, err := natsserver.NewServer(
		&natsserver.Options{DontListen: true, JetStream: true, StoreDir: t.TempDir()},
	)
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}
	go broker.Start()
	if !broker.ReadyForConnections(callbackWait) {
		t.Fatal("broker not ready")
	}
	t.Cleanup(broker.Shutdown)

	js, connection, err := jetstreamconnect.Open(
		"nats://broker.internal:4222",
		func(context.Context, string, string) (net.Conn, error) { return broker.InProcessConn() },
		jetstreamconnect.UnobservedConnection{},
		jetstreamconnect.BrokerAccess{},
	)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer connection.Close()
	if _, err := js.AccountInfo(context.Background()); err != nil {
		t.Errorf("account info through the dialer: %v", err)
	}
}

func TestOpenRejectsUnreachable(t *testing.T) {
	_, _, err := jetstreamconnect.Open(
		"nats://127.0.0.1:1", (&net.Dialer{}).DialContext,
		jetstreamconnect.UnobservedConnection{}, jetstreamconnect.BrokerAccess{},
	)
	if !errors.Is(err, jetstreamconnect.ErrBrokerUnreachable) {
		t.Fatalf("err = %v, want ErrBrokerUnreachable", err)
	}
}

type connectionEvents chan string

func (events connectionEvents) ConnectionLost()     { events <- "lost" }
func (events connectionEvents) ConnectionRestored() { events <- "restored" }

func brokerOn(t *testing.T, port int) *natsserver.Server {
	t.Helper()
	broker, err := natsserver.NewServer(&natsserver.Options{Host: "127.0.0.1", Port: port})
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}
	go broker.Start()
	if !broker.ReadyForConnections(callbackWait) {
		t.Fatal("broker not ready")
	}
	t.Cleanup(broker.Shutdown)
	return broker
}

func awaitEvent(t *testing.T, events connectionEvents, want string) {
	t.Helper()
	select {
	case event := <-events:
		if event != want {
			t.Fatalf("event = %q, want %q", event, want)
		}
	case <-time.After(callbackWait):
		t.Fatalf("no %s event", want)
	}
}

func TestOpenReportsALostAndARestoredConnection(t *testing.T) {
	broker := brokerOn(t, -1)
	port := broker.Addr().(*net.TCPAddr).Port
	events := make(connectionEvents, 2)
	_, connection, err := jetstreamconnect.Open(
		broker.ClientURL(),
		(&net.Dialer{}).DialContext,
		events,
		jetstreamconnect.BrokerAccess{},
	)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer connection.Close()

	broker.Shutdown()
	awaitEvent(t, events, "lost")
	brokerOn(t, port)
	awaitEvent(t, events, "restored")
}

type namedClientCheck struct {
	names chan string
}

func (c namedClientCheck) Check(client natsserver.ClientAuthentication) bool {
	options := client.GetOpts()
	c.names <- options.Name
	return options.Password == "secret"
}

func TestOpenGivesTheCredentialsOfTheAccess(t *testing.T) {
	check := namedClientCheck{names: make(chan string, 1)}
	broker, err := natsserver.NewServer(&natsserver.Options{
		Host: "127.0.0.1", Port: -1, CustomClientAuthentication: check,
	})
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}
	go broker.Start()
	if !broker.ReadyForConnections(callbackWait) {
		t.Fatal("broker not ready")
	}
	t.Cleanup(broker.Shutdown)

	_, connection, err := jetstreamconnect.Open(
		broker.ClientURL(),
		(&net.Dialer{}).DialContext,
		jetstreamconnect.UnobservedConnection{},
		jetstreamconnect.BrokerAccess{ClientName: "Pixel 8", Password: "secret"},
	)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer connection.Close()

	if name := <-check.names; name != "Pixel 8" {
		t.Errorf("client name = %q, want Pixel 8", name)
	}
}

func TestOpenTakesRepliesUnderTheInboxPrefixOfTheAccess(t *testing.T) {
	broker := brokerOn(t, -1)
	_, connection, err := jetstreamconnect.Open(
		broker.ClientURL(),
		(&net.Dialer{}).DialContext,
		make(connectionEvents, 1),
		jetstreamconnect.BrokerAccess{InboxPrefix: "_INBOX.d7f3"},
	)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer connection.Close()

	if inbox := connection.NewRespInbox(); !strings.HasPrefix(inbox, "_INBOX.d7f3.") {
		t.Errorf("reply inbox = %q, want one under _INBOX.d7f3", inbox)
	}
}

func TestOpenConnectsOverTheTLSOfTheAccess(t *testing.T) {
	certificate := selfSignedCertificate(t)
	broker := tlsBrokerWith(t, certificate)
	_, connection, err := jetstreamconnect.Open(
		"tls://"+broker.Addr().String(),
		(&net.Dialer{}).DialContext,
		make(connectionEvents, 1),
		jetstreamconnect.BrokerAccess{TLS: &tls.Config{
			RootCAs:    certificatePoolOf(t, certificate),
			ServerName: "127.0.0.1",
			MinVersion: tls.VersionTLS13,
		}},
	)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer connection.Close()

	if _, err := connection.TLSConnectionState(); err != nil {
		t.Errorf("TLS state: %v, want a TLS connection", err)
	}
}

func selfSignedCertificate(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		IsCA:         true,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},

		BasicConstraintsValid: true,
	}
	certificateDER, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&key.PublicKey,
		key,
	)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{certificateDER}, PrivateKey: key}
}

func tlsBrokerWith(t *testing.T, certificate tls.Certificate) *natsserver.Server {
	t.Helper()
	broker, err := natsserver.NewServer(&natsserver.Options{
		Host: "127.0.0.1",
		Port: -1,
		TLS:  true,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{certificate},
			MinVersion:   tls.VersionTLS13,
		},
	})
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}
	go broker.Start()
	if !broker.ReadyForConnections(callbackWait) {
		t.Fatal("broker not ready")
	}
	t.Cleanup(broker.Shutdown)
	return broker
}

func certificatePoolOf(t *testing.T, certificate tls.Certificate) *x509.CertPool {
	t.Helper()
	parsedCertificate, err := x509.ParseCertificate(certificate.Certificate[0])
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(parsedCertificate)
	return pool
}
