package companionrequests

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

type RecordableDevices interface {
	RecordAppVersion(ctx context.Context, deviceID companionprotocol.DeviceID, appVersion string)
}

type Unpairer interface {
	Unpair(ctx context.Context, deviceID companionprotocol.DeviceID)
}

type DeviceAnswerer struct {
	devices  RecordableDevices
	unpairer Unpairer
	settings companionprotocol.DeviceSettings
}

func NewDeviceAnswerer(
	devices RecordableDevices,
	unpairer Unpairer,
	settings companionprotocol.DeviceSettings,
) *DeviceAnswerer {
	return &DeviceAnswerer{devices: devices, unpairer: unpairer, settings: settings}
}

func (d *DeviceAnswerer) Subscribe(
	ctx context.Context,
	connection *nats.Conn,
) (*Subscriptions, error) {
	return subscribeTo(connection, map[string]nats.MsgHandler{
		companionprotocol.SettingsSubjects: func(message *nats.Msg) {
			respond(ctx, message, d.settings)
		},
		companionprotocol.AppReportSubjects: func(message *nats.Msg) {
			d.recordAppReport(ctx, message)
		},
		companionprotocol.UnpairingRequestSubjects: func(message *nats.Msg) {
			d.unpair(ctx, message, connection)
		},
	})
}

func (d *DeviceAnswerer) recordAppReport(ctx context.Context, message *nats.Msg) {
	var report companionprotocol.AppReport
	if err := json.Unmarshal(message.Data, &report); err != nil {
		slog.WarnContext(ctx, "unreadable app report",
			slog.String("subject", message.Subject), slog.Any("error", err),
		)
		return
	}
	d.devices.RecordAppVersion(
		ctx,
		companionprotocol.DeviceIDIn(message.Subject),
		report.AppVersion,
	)
}

func (d *DeviceAnswerer) unpair(ctx context.Context, message *nats.Msg, connection *nats.Conn) {
	deviceID := companionprotocol.DeviceIDIn(message.Subject)
	confirmUnpairing(ctx, deviceID, message, connection)
	d.unpairer.Unpair(ctx, deviceID)
	reportUnpairedItself(ctx, deviceID)
}

func confirmUnpairing(
	ctx context.Context,
	deviceID companionprotocol.DeviceID,
	message *nats.Msg,
	connection *nats.Conn,
) {
	err := message.Respond(nil)
	if err == nil {
		err = connection.Flush()
	}
	if err != nil {
		slog.WarnContext(ctx, "unpairing not confirmed to the device",
			slog.String("device", string(deviceID)), slog.Any("error", err),
		)
	}
}

func reportUnpairedItself(ctx context.Context, deviceID companionprotocol.DeviceID) {
	slog.DebugContext(ctx, "device unpaired itself", slog.String("device", string(deviceID)))
}
