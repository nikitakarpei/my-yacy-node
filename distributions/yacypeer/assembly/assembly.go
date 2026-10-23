package yacypeerassembly

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
)

const brokerDirectoryName = "nats"

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	ops net.Listener,
	getenv func(string) string,
) error {
	broker, err := natsbroker.Start(cfg.directoryOf(brokerDirectoryName))
	if err != nil {
		return errors.Join(err, ops.Close())
	}
	defer broker.Shutdown()

	members, err := membersFor(ctx, cfg, broker.ClientURL(), getenv)
	if err != nil {
		return errors.Join(err, ops.Close())
	}
	logStarted(ctx, broker, ops)
	opsMux := opsMuxFor(opsmetrics.ProcessRegistry(), members)
	return opsmetrics.ServeAlongside(ctx, ops, opsMux, func(ctx context.Context) error {
		return runEvery(ctx, members)
	})
}

func logStarted(ctx context.Context, broker *natsbroker.Broker, ops net.Listener) {
	slog.DebugContext(ctx, "yacypeer started",
		slog.String("natsUrl", broker.ClientURL()),
		slog.String("opsAddr", ops.Addr().String()),
	)
}

func opsMuxFor(processRegistry *prometheus.Registry, members []member) *http.ServeMux {
	mux := opsmetrics.MuxFor(processRegistry)
	for _, member := range members {
		mux.Handle(opsmetrics.PathMetrics+"/"+member.name, opsmetrics.HandlerFor(member.registry))
	}
	return mux
}

func runEvery(ctx context.Context, members []member) error {
	runs := make([]func(context.Context) error, 0, len(members))
	for _, member := range members {
		runs = append(runs, member.run)
	}
	return servergroup.Run(ctx, 0, nil, runs...)
}
