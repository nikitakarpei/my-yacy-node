package companionaccess_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/natspermissions"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/brokerkey"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionaccess"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
)

const (
	workSubject  = "work.done"
	pairingWait  = 30 * time.Second
	violationMax = 5 * time.Second
)

var connectedAt = time.Now().Truncate(time.Second)

type fixedClock struct{}

func (fixedClock) Now() time.Time { return connectedAt }

type knownDevices struct {
	paired    map[companionprotocol.DeviceID]bool
	connected map[companionprotocol.DeviceID]time.Time
	mu        sync.Mutex
}

func (d *knownDevices) Holds(id companionprotocol.DeviceID) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.paired[id]
}

func (d *knownDevices) RecordConnection(
	_ context.Context,
	id companionprotocol.DeviceID,
	at time.Time,
) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.connected[id] = at
}

type openSecrets struct {
	begun map[companionprotocol.Secret]companionprotocol.DeviceName
	open  map[companionprotocol.Secret]bool
	mu    sync.Mutex
}

func (s *openSecrets) Begin(
	_ context.Context,
	secret companionprotocol.Secret,
	deviceName companionprotocol.DeviceName,
) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.open[secret] {
		return false
	}
	delete(s.open, secret)
	s.begun[secret] = deviceName
	return true
}

type fixture struct {
	broker  *natsbroker.Broker
	devices *knownDevices
	secrets *openSecrets
}

func startBroker(t *testing.T) fixture {
	t.Helper()
	return startBrokerFor(t, natspermissions.PublishingOn(workSubject))
}

func startBrokerFor(t *testing.T, pageServicePermissions natspermissions.Permissions) fixture {
	t.Helper()
	devices := &knownDevices{
		paired:    map[companionprotocol.DeviceID]bool{},
		connected: map[companionprotocol.DeviceID]time.Time{},
	}
	secrets := &openSecrets{
		begun: map[companionprotocol.Secret]companionprotocol.DeviceName{},
		open:  map[companionprotocol.Secret]bool{},
	}
	certificate, err := brokerkey.Settle(t.TempDir())
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	broker, err := natsbroker.StartListening(t.TempDir(), natsbroker.Listening{
		Authentication: companionaccess.New(
			devices, secrets, fixedClock{}, pageServicePermissions, pairingWait,
		),
		Address:     "127.0.0.1:0",
		Certificate: certificate,
	})
	if err != nil {
		t.Fatalf("StartListening: %v", err)
	}
	t.Cleanup(broker.Shutdown)
	return fixture{broker: broker, devices: devices, secrets: secrets}
}

func deviceCertificate(t *testing.T) (tls.Certificate, companionprotocol.DeviceID) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    connectedAt.Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key},
		companionprotocol.DeviceIDOf(leaf.RawSubjectPublicKeyInfo)
}

type violations chan error

func connect(
	t *testing.T,
	broker *natsbroker.Broker,
	certificates []tls.Certificate,
	options ...nats.Option,
) (*nats.Conn, violations, error) {
	t.Helper()
	seen := make(violations, 8)
	connection, err := nats.Connect(
		"tls://"+broker.ListeningAddress().String(),
		append(options,
			nats.Secure(&tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // the test pins nothing.
				Certificates:       certificates,
			}),
			nats.MaxReconnects(0),
			nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
				seen <- err
			}),
		)...,
	)
	if err != nil {
		return nil, seen, fmt.Errorf("connect: %w", err)
	}
	t.Cleanup(connection.Close)
	return connection, seen, nil
}

func requireAllowed(t *testing.T, connection *nats.Conn, seen violations, subject string) {
	t.Helper()
	if err := connection.Publish(subject, nil); err != nil {
		t.Fatalf("publish %s: %v", subject, err)
	}
	if err := connection.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	select {
	case err := <-seen:
		t.Errorf("publish %s: %v", subject, err)
	case <-time.After(100 * time.Millisecond):
	}
}

func requireRefused(t *testing.T, connection *nats.Conn, seen violations, subject string) {
	t.Helper()
	if err := connection.Publish(subject, nil); err != nil {
		t.Fatalf("publish %s: %v", subject, err)
	}
	select {
	case err := <-seen:
		if !errors.Is(err, nats.ErrPermissionViolation) {
			t.Errorf("publish %s: %v, want a permission violation", subject, err)
		}
	case <-time.After(violationMax):
		t.Errorf("publish %s passed", subject)
	}
}

func TestAPairedDeviceConnectsWithItsKeyAndMayDoItsWork(t *testing.T) {
	broker := startBroker(t)
	certificate, device := deviceCertificate(t)
	broker.devices.paired[device] = true

	connection, seen, err := connect(t, broker.broker, []tls.Certificate{certificate},
		nats.CustomInboxPrefix(companionprotocol.DeviceInboxPrefixOf(device)),
	)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	requireAllowed(t, connection, seen, workSubject)
	requireAllowed(t, connection, seen, companionprotocol.SettingsSubjectOf(device))
	requireRefused(t, connection, seen, companionprotocol.SettingsSubjectOf("other-device"))
	if !broker.broker.ConnectedUsers()[companionaccess.UserOf(device)] {
		t.Error("the device is not connected as its user")
	}
	if !broker.devices.connected[device].Equal(connectedAt) {
		t.Errorf(
			"connection recorded at %v, want %v",
			broker.devices.connected[device],
			connectedAt,
		)
	}
}

func TestAnUnpairedDeviceCannotConnect(t *testing.T) {
	broker := startBroker(t)
	certificate, _ := deviceCertificate(t)

	if _, _, err := connect(t, broker.broker, []tls.Certificate{certificate}); err == nil {
		t.Error("an unpaired device connected")
	}
}

func TestADeviceWithAnOpenSecretMayOnlyAskToPair(t *testing.T) {
	broker := startBroker(t)
	secret := companionprotocol.NewSecret()
	broker.secrets.open[secret] = true

	connection, seen, err := connect(t, broker.broker, nil,
		nats.Name("Pixel 8"),
		nats.UserInfo("", string(secret)),
		nats.CustomInboxPrefix(companionprotocol.PairingInboxPrefixFor(secret)),
	)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	if broker.secrets.begun[secret] != "Pixel 8" {
		t.Errorf("begun pairings = %v, want Pixel 8", broker.secrets.begun)
	}
	requireAllowed(t, connection, seen, companionprotocol.PairingRequestSubject)
	requireRefused(t, connection, seen, workSubject)
}

func TestADeviceWithoutAnOpenSecretCannotConnect(t *testing.T) {
	broker := startBroker(t)

	_, _, err := connect(
		t,
		broker.broker,
		nil,
		nats.UserInfo("", string(companionprotocol.NewSecret())),
	)
	if err == nil {
		t.Error("a device without an open secret connected")
	}
}

func TestAConnectionWithoutTLSIsRefused(t *testing.T) {
	broker := startBroker(t)

	if _, err := nats.Connect(
		"nats://"+broker.broker.ListeningAddress().String(),
		nats.MaxReconnects(0),
	); err == nil {
		t.Error("a connection without TLS connected")
	}
}
