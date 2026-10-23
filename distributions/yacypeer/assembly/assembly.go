package yacypeerassembly

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
)

const (
	brokerDirectoryName = "nats"
	metricsPathPrefix   = "/metrics/"
)

type Listeners struct {
	Ops net.Listener
}

func ListenersFor(ctx context.Context, cfg ServiceConfig) (Listeners, error) {
	ops, err := servergroup.Listen(ctx, cfg.OpsAddr)
	if err != nil {
		return Listeners{}, err
	}
	return Listeners{Ops: ops}, nil
}

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	listeners Listeners,
	getenv func(string) string,
) error {
	broker, err := natsbroker.Start(cfg.directoryOf(brokerDirectoryName))
	if err != nil {
		return errors.Join(err, listeners.Ops.Close())
	}
	defer broker.Shutdown()

	members, err := membersFor(ctx, cfg, broker.ClientURL(), getenv)
	if err != nil {
		return errors.Join(err, listeners.Ops.Close())
	}
	slog.InfoContext(ctx, "yacypeer started",
		slog.String("natsUrl", broker.ClientURL()),
		slog.String("opsAddr", listeners.Ops.Addr().String()),
	)
	opsMux := opsMuxFor(members)
	return opsmetrics.ServeAlongside(ctx, listeners.Ops, opsMux, func(ctx context.Context) error {
		return runEvery(ctx, members)
	})
}

func opsMuxFor(members []member) *http.ServeMux {
	mux := http.NewServeMux()
	for _, member := range members {
		mux.Handle(
			metricsPathPrefix+member.name,
			promhttp.HandlerFor(member.registry, promhttp.HandlerOpts{}),
		)
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
