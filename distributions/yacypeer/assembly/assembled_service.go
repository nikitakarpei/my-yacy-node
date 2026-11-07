package yacypeerassembly

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/serviceenvironment"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/servicelog"
)

type dial = func(ctx context.Context, network, address string) (net.Conn, error)

type assembledService[C any, L io.Closer] struct {
	cfg        C
	listeners  L
	runService func(context.Context, C, L, dial, *prometheus.Registry) error
	dial       dial
	name       string
}

func assembledServiceFrom[C any, L io.Closer](
	ctx context.Context,
	name string,
	wiring serviceenvironment.Wiring,
	loadServiceConfig func(func(string) string) (C, error),
	listenersFor func(context.Context, C) (L, error),
	runService func(context.Context, C, L, dial, *prometheus.Registry) error,
	dial dial,
	getenv func(string) string,
) (assembledService[C, L], error) {
	environment, err := serviceenvironment.For(
		name,
		wiring,
		unreadSettings,
		getenv,
	)
	if err != nil {
		return assembledService[C, L]{}, err
	}
	cfg, err := loadServiceConfig(environment.Lookup)
	if err != nil {
		return assembledService[C, L]{}, fmt.Errorf("%s: %w", name, err)
	}
	listeners, err := listenersFor(ctx, cfg)
	if err != nil {
		return assembledService[C, L]{}, fmt.Errorf("%s: %w", name, err)
	}
	return assembledService[C, L]{
		cfg:        cfg,
		listeners:  listeners,
		runService: runService,
		dial:       dial,
		name:       name,
	}, nil
}

func (a assembledService[C, L]) service() service {
	registry := prometheus.NewRegistry()
	return service{
		registry:  registry,
		listeners: a.listeners,
		run: func(ctx context.Context) error {
			return a.runService(
				servicelog.ContextFor(ctx, a.name),
				a.cfg,
				a.listeners,
				a.dial,
				registry,
			)
		},
		name: a.name,
	}
}
