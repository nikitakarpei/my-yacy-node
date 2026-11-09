package companionrequests

import (
	"context"
	"encoding/json"
	"fmt"
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
		companionprotocol.HelloSubjects: func(message *nats.Msg) {
			d.answerHello(ctx, message)
		},
		companionprotocol.SettingsSubjects: func(message *nats.Msg) {
			respond(ctx, message, d.settings)
		},
		companionprotocol.UnpairingRequestSubjects: func(message *nats.Msg) {
			d.unpair(ctx, message, connection)
		},
	})
}

func (d *DeviceAnswerer) answerHello(ctx context.Context, message *nats.Msg) {
	deviceID := companionprotocol.DeviceIDIn(message.Subject)
	hello, refusal := helloFrom(message.Data)
	if refusal != nil {
		refuseHello(ctx, message, deviceID, refusal)
		return
	}
	d.devices.RecordAppVersion(ctx, deviceID, hello.AppVersion)
	respond(ctx, message, companionprotocol.AcceptedHelloReply())
}

func helloFrom(data []byte) (companionprotocol.Hello, *requestRefusal) {
	var hello companionprotocol.Hello
	if err := json.Unmarshal(data, &hello); err != nil {
		return hello, &requestRefusal{
			reason: companionprotocol.ReasonRequestInvalid,
			cause:  fmt.Errorf("read the hello: %w", err),
		}
	}
	return hello, protocolRefusalOf(hello.ProtocolVersion)
}

func refuseHello(
	ctx context.Context,
	message *nats.Msg,
	deviceID companionprotocol.DeviceID,
	refusal *requestRefusal,
) {
	slog.WarnContext(ctx, "hello of a device refused",
		slog.String("device", string(deviceID)),
		slog.String("reason", string(refusal.reason)),
		slog.Any("cause", refusal.cause),
	)
	respond(ctx, message, companionprotocol.RefusedHelloReply(refusal.reason))
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
