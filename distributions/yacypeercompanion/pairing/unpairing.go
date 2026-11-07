package pairing

import (
	"context"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/brokeraccess"
)

const unpairWait = 5 * time.Second

func Unpair(linkText string, key DeviceKey) {
	link, err := companionprotocol.PairLinkFrom(linkText)
	if err != nil {
		return
	}
	options, err := brokeraccess.DeviceOptions(link, key)
	if err != nil {
		slog.WarnContext(context.Background(), "unpair request not sent", slog.Any("error", err))
		return
	}
	connection, err := brokeraccess.Connect(link, append(options, nats.NoReconnect()))
	if err != nil {
		slog.WarnContext(
			context.Background(),
			"peer unreachable for unpairing",
			slog.Any("error", err),
		)
		return
	}
	defer connection.Close()
	device := companionprotocol.DeviceIDOf(key.PublicKeyInfo())
	if _, err := connection.Request(
		companionprotocol.UnpairSubjectOf(device), nil, unpairWait,
	); err != nil {
		slog.WarnContext(context.Background(), "no unpair reply", slog.Any("error", err))
	}
}
