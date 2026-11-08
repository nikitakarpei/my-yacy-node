// Package companionaccess decides which devices connect to the endpoint of the broker of the peer
// and what each connection may do. A paired device connects with the key that it paired with and
// may run the page services and use its own subjects. A device that pairs connects with the
// secret of an open pairing page and may only ask to pair.
package companionaccess

import (
	"context"
	"crypto/x509"
	"log/slog"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/natspermissions"
)

const (
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

type Gatekeeper struct {
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
) *Gatekeeper {
	return &Gatekeeper{
		devices:                 devices,
		offers:                  offers,
		clock:                   clock,
		pageServicesPermissions: pageServicesPermissions,
		pairingWait:             pairingWait,
	}
}

func DeviceUserOf(deviceID companionprotocol.DeviceID) string {
	return deviceUserStart + string(deviceID)
}

func (g *Gatekeeper) Check(client natsserver.ClientAuthentication) bool {
	if certificates := client.GetTLSConnectionState().PeerCertificates; len(certificates) > 0 {
		return g.admitPairedDevice(client, certificates[0])
	}
	return g.admitPairingDevice(client)
}

func (g *Gatekeeper) admitPairedDevice(
	client natsserver.ClientAuthentication,
	certificate *x509.Certificate,
) bool {
	ctx := context.Background()
	deviceID := companionprotocol.DeviceIDFrom(certificate.RawSubjectPublicKeyInfo)
	if !g.devices.Holds(deviceID) {
		slog.WarnContext(ctx, "unpaired device refused",
			slog.String("device", string(deviceID)),
			slog.String("remoteAddress", client.RemoteAddress().String()),
		)
		return false
	}
	client.RegisterUser(&natsserver.User{
		Username:    DeviceUserOf(deviceID),
		Permissions: g.devicePermissionsFor(deviceID),
	})
	g.devices.RecordConnection(ctx, deviceID, g.clock.Now())
	slog.DebugContext(ctx, "paired device connected", slog.String("device", string(deviceID)))
	return true
}

func (g *Gatekeeper) devicePermissionsFor(
	deviceID companionprotocol.DeviceID,
) *natsserver.Permissions {
	return &natsserver.Permissions{
		Publish: &natsserver.SubjectPermission{
			Allow: append(
				append([]string{}, g.pageServicesPermissions.Publish...),
				companionprotocol.DeviceSubjectsOf(deviceID),
			),
		},
		Subscribe: &natsserver.SubjectPermission{
			Allow: append(
				append([]string{}, g.pageServicesPermissions.Subscribe...),
				companionprotocol.DeviceInboxSubjectsOf(deviceID),
			),
		},
	}
}

func (g *Gatekeeper) admitPairingDevice(client natsserver.ClientAuthentication) bool {
	ctx := context.Background()
	options := client.GetOpts()
	credentials := companionprotocol.PairingCredentials{
		ClientName: options.Name,
		Password:   options.Password,
	}
	if !g.offers.Begin(ctx, credentials.Secret(), credentials.DeviceName()) {
		slog.WarnContext(ctx, "pairing device refused",
			slog.String("deviceName", string(credentials.DeviceName())),
			slog.String("remoteAddress", client.RemoteAddress().String()),
		)
		return false
	}
	client.RegisterUser(&natsserver.User{
		Username:           pairingUserOf(credentials.Secret()),
		Permissions:        pairingPermissionsFor(credentials.Secret()),
		ConnectionDeadline: g.clock.Now().Add(g.pairingWait),
	})
	slog.DebugContext(ctx, "device began pairing",
		slog.String("deviceName", string(credentials.DeviceName())),
	)
	return true
}

func pairingUserOf(secret companionprotocol.Secret) string {
	return pairingUserStart + string(companionprotocol.SecretIDFrom(secret))
}

func pairingPermissionsFor(secret companionprotocol.Secret) *natsserver.Permissions {
	return &natsserver.Permissions{
		Publish: &natsserver.SubjectPermission{
			Allow: []string{companionprotocol.PairingRequestSubject},
		},
		Subscribe: &natsserver.SubjectPermission{
			Allow: []string{companionprotocol.PairingInboxSubjectsFor(secret)},
		},
	}
}
