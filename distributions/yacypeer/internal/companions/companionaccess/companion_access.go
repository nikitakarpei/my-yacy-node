// Package companionaccess decides who connects to the broker of the peer and what each
// connection may do. The parts of the peer connect inside the process and may do anything. A
// paired device connects with the key that it paired with and may run the page services and use
// its own subjects. A device that pairs connects with the secret of an open pairing page and may
// only ask to pair.
package companionaccess

import (
	"context"
	"crypto/x509"
	"log/slog"
	"strings"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/natspermissions"
)

const (
	inProcessNetwork = "pipe"
	deviceUserStart  = "device-"
	pairingUserStart = "pairing-"
)

type PairedDevices interface {
	Holds(id companionprotocol.DeviceID) bool
	RecordConnection(ctx context.Context, id companionprotocol.DeviceID, at time.Time)
}

type PairingOffers interface {
	Begin(
		ctx context.Context,
		secret companionprotocol.Secret,
		deviceName companionprotocol.DeviceName,
	) bool
}

type Clock interface {
	Now() time.Time
}

type Access struct {
	devices                 PairedDevices
	offers                  PairingOffers
	clock                   Clock
	pageServicesPermissions natspermissions.Permissions
	pairingWait             time.Duration
}

func New(
	devices PairedDevices,
	offers PairingOffers,
	clock Clock,
	pageServicesPermissions natspermissions.Permissions,
	pairingWait time.Duration,
) *Access {
	return &Access{
		devices:                 devices,
		offers:                  offers,
		clock:                   clock,
		pageServicesPermissions: pageServicesPermissions,
		pairingWait:             pairingWait,
	}
}

func UserOf(device companionprotocol.DeviceID) string {
	return deviceUserStart + string(device)
}

func (a *Access) Check(client natsserver.ClientAuthentication) bool {
	if address := client.RemoteAddress(); address != nil &&
		address.Network() == inProcessNetwork {
		return true
	}
	state := client.GetTLSConnectionState()
	if state == nil {
		slog.WarnContext(context.Background(), "connection without TLS refused",
			slog.String("remoteAddress", client.RemoteAddress().String()),
		)
		return false
	}
	if len(state.PeerCertificates) > 0 {
		return a.checkDevice(client, state.PeerCertificates[0])
	}
	return a.checkPairing(client)
}

func (a *Access) checkDevice(
	client natsserver.ClientAuthentication,
	certificate *x509.Certificate,
) bool {
	ctx := context.Background()
	device := companionprotocol.DeviceIDOf(certificate.RawSubjectPublicKeyInfo)
	if !a.devices.Holds(device) {
		slog.WarnContext(ctx, "unpaired device refused",
			slog.String("device", string(device)),
			slog.String("remoteAddress", client.RemoteAddress().String()),
		)
		return false
	}
	client.RegisterUser(&natsserver.User{
		Username:    UserOf(device),
		Permissions: a.permissionsOf(device),
	})
	a.devices.RecordConnection(ctx, device, a.clock.Now())
	slog.DebugContext(ctx, "paired device connected", slog.String("device", string(device)))
	return true
}

func (a *Access) permissionsOf(device companionprotocol.DeviceID) *natsserver.Permissions {
	return &natsserver.Permissions{
		Publish: &natsserver.SubjectPermission{
			Allow: append(
				append([]string{}, a.pageServicesPermissions.Publish...),
				companionprotocol.DeviceSubjectsOf(device),
			),
		},
		Subscribe: &natsserver.SubjectPermission{
			Allow: append(
				append([]string{}, a.pageServicesPermissions.Subscribe...),
				companionprotocol.DeviceRepliesOf(device),
			),
		},
	}
}

func (a *Access) checkPairing(client natsserver.ClientAuthentication) bool {
	ctx := context.Background()
	options := client.GetOpts()
	secret := companionprotocol.Secret(options.Password)
	if !a.offers.Begin(ctx, secret, companionprotocol.DeviceName(options.Name)) {
		slog.WarnContext(ctx, "pairing device refused",
			slog.String("deviceName", options.Name),
			slog.String("remoteAddress", client.RemoteAddress().String()),
		)
		return false
	}
	client.RegisterUser(&natsserver.User{
		Username:           pairingUserStart + string(companionprotocol.SecretIDOf(secret)),
		Permissions:        pairingPermissionsFor(secret),
		ConnectionDeadline: a.clock.Now().Add(a.pairingWait),
	})
	slog.DebugContext(ctx, "device began pairing", slog.String("deviceName", options.Name))
	return true
}

func pairingPermissionsFor(secret companionprotocol.Secret) *natsserver.Permissions {
	return &natsserver.Permissions{
		Publish: &natsserver.SubjectPermission{
			Allow: []string{companionprotocol.PairingRequestSubject},
		},
		Subscribe: &natsserver.SubjectPermission{
			Allow: []string{companionprotocol.PairingRepliesFor(secret)},
		},
	}
}

func IsDeviceUser(user string) bool {
	return strings.HasPrefix(user, deviceUserStart)
}
