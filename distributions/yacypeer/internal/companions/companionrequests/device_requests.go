package companionrequests

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

type ReportingDevices interface {
	RecordAppVersion(ctx context.Context, id companionprotocol.DeviceID, appVersion string)
}

type Unpairer interface {
	Unpair(ctx context.Context, device companionprotocol.DeviceID)
}

type DeviceRequests struct {
	devices  ReportingDevices
	unpairer Unpairer
	settings companionprotocol.Settings
}

func NewDeviceRequests(
	devices ReportingDevices,
	unpairer Unpairer,
	settings companionprotocol.Settings,
) *DeviceRequests {
	return &DeviceRequests{devices: devices, unpairer: unpairer, settings: settings}
}

func (d *DeviceRequests) Serve(ctx context.Context, connection *nats.Conn) error {
	return answerUntilDone(ctx, connection, map[string]nats.MsgHandler{
		companionprotocol.SettingsSubjects: func(request *nats.Msg) {
			respond(ctx, request, d.settings)
		},
		companionprotocol.AppSubjects: func(report *nats.Msg) {
			d.recordApp(ctx, report)
		},
		companionprotocol.UnpairSubjects: func(request *nats.Msg) {
			d.unpair(ctx, request, connection)
		},
	})
}

func (d *DeviceRequests) recordApp(ctx context.Context, report *nats.Msg) {
	var app companionprotocol.AppReport
	if err := json.Unmarshal(report.Data, &app); err != nil {
		slog.WarnContext(ctx, "unreadable app report",
			slog.String("subject", report.Subject), slog.Any("error", err),
		)
		return
	}
	d.devices.RecordAppVersion(ctx, companionprotocol.DeviceFromSubject(report.Subject),
		app.AppVersion)
}

func (d *DeviceRequests) unpair(ctx context.Context, request *nats.Msg, connection *nats.Conn) {
	device := companionprotocol.DeviceFromSubject(request.Subject)
	confirmUnpairing(ctx, device, request, connection)
	d.unpairer.Unpair(ctx, device)
	slog.DebugContext(ctx, "device unpaired itself", slog.String("device", string(device)))
}

func confirmUnpairing(
	ctx context.Context,
	device companionprotocol.DeviceID,
	request *nats.Msg,
	connection *nats.Conn,
) {
	err := request.Respond(nil)
	if err == nil {
		err = connection.Flush()
	}
	if err != nil {
		slog.WarnContext(ctx, "unpairing not confirmed to the device",
			slog.String("device", string(device)), slog.Any("error", err),
		)
	}
}
