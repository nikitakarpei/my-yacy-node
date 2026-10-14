package servergroup

import (
	"context"
	"fmt"
	"net"
)

func Listen(ctx context.Context, addr string) (net.Listener, error) {
	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}

	return listener, nil
}
