package companionaccess_test

import (
	"context"
	"crypto/tls"
	"maps"
	"net"
	"testing"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionaccess"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionaccess/companionaccesstest"
)

func TestDeviceConnectionsNameOnlyTheConnectedPairedDevices(t *testing.T) {
	fixture := startFixture(t)
	certificate, deviceID := companionaccesstest.DeviceCertificate(t)
	fixture.devices.paired[deviceID] = true
	if _, _, err := connect(t, fixture.broker, []tls.Certificate{certificate}); err != nil {
		t.Fatalf("connect the paired device: %v", err)
	}
	secret := companionprotocol.RandomSecret()
	fixture.secrets.open[secret] = true
	if _, _, err := connect(t, fixture.broker, nil, nats.UserInfo("", string(secret))); err != nil {
		t.Fatalf("connect the pairing device: %v", err)
	}
	_, inProcess, err := jetstreamconnect.Open(
		"nats://broker.internal",
		func(context.Context, string, string) (net.Conn, error) { return fixture.broker.Connection() },
		jetstreamconnect.UnobservedConnection{},
		jetstreamconnect.BrokerAccess{},
	)
	if err != nil {
		t.Fatalf("connect in the process: %v", err)
	}
	defer inProcess.Close()

	connected := companionaccess.NewDeviceConnections(fixture.broker).ConnectedDevices()

	want := map[companionprotocol.DeviceID]bool{deviceID: true}
	if !maps.Equal(connected, want) {
		t.Errorf("connected devices = %v, want %v", connected, want)
	}
}
