package pairing

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/brokeraccess"
)

const unpairWait = 5 * time.Second

func Unpair(linkText string, key DeviceKey) {
	link, err := companionprotocol.PairLinkFrom(linkText)
	if err != nil {
		slog.WarnContext(context.Background(), "unpairing request not sent", slog.Any("error", err))
		return
	}
	access, err := brokeraccess.DeviceAccess(link, key)
	if err != nil {
		slog.WarnContext(context.Background(), "unpairing request not sent", slog.Any("error", err))
		return
	}
	connection, err := brokeraccess.NewConnector(wallclock.Clock{}).ConnectOnce(link, access)
	if err != nil {
		slog.WarnContext(
			context.Background(),
			"peer unreachable for unpairing",
			slog.Any("error", err),
		)
		return
	}
	defer connection.Close()
	deviceID := companionprotocol.DeviceIDFrom(key.PublicKeyInfo())
	if _, err := connection.Request(
		companionprotocol.UnpairingRequestSubjectOf(deviceID), nil, unpairWait,
	); err != nil {
		slog.WarnContext(context.Background(), "no unpairing reply", slog.Any("error", err))
	}
}
