// Package loopback opens the listeners of the peer that only the local host can reach.
package loopback

import (
	"context"
	"net"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

const (
	Host           = "127.0.0.1"
	AnyFreeAddress = Host + ":0"
)

func Listen(ctx context.Context) (net.Listener, error) {
	return servergroup.Listen(ctx, AnyFreeAddress)
}
