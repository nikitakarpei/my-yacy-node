package yacypeerassembly

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	corpusmarkdownassembly "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/assembly"
	corpustextassembly "github.com/nikitakarpei/yacy-rwi-node/corpustext/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	yacydhtsearchassembly "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/assembly"
	yacynodeassembly "github.com/nikitakarpei/yacy-rwi-node/yacynode/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/memberenvironment"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/memberlog"
)

const (
	yacynodeMember       = "yacynode"
	yacydhtsearchMember  = "yacydhtsearch"
	corpustextMember     = "corpustext"
	corpusmarkdownMember = "corpusmarkdown"
)

type member struct {
	registry *prometheus.Registry
	run      func(context.Context) error
	name     string
}

type memberAssembly interface {
	memberFor(ctx context.Context, getenv func(string) string) (member, error)
}

func membersFor(
	ctx context.Context,
	cfg ServiceConfig,
	natsURL string,
	getenv func(string) string,
) ([]member, error) {
	assemblies := memberAssembliesFor(cfg, natsURL)
	members := make([]member, 0, len(assemblies))
	for _, assembly := range assemblies {
		member, err := assembly.memberFor(ctx, getenv)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, nil
}

func memberAssembliesFor(cfg ServiceConfig, natsURL string) []memberAssembly {
	return []memberAssembly{
		memberAssemblyOf(
			yacynodeMember,
			memberenvironment.Wiring{
				yacynodeassembly.EnvPageOfferNATSURL:  natsURL,
				yacynodeassembly.EnvPageOfferDurable:  yacynodeMember,
				yacynodeassembly.EnvAcceptRemoteIndex: strconv.FormatBool(false),
				yacynodeassembly.EnvDataDir:           cfg.directoryOf(yacynodeMember),
				yacynodeassembly.EnvNetworkName:       cfg.NetworkName,
				yacynodeassembly.EnvSeedlistURLs:      cfg.SeedlistURLs,
				yacynodeassembly.EnvOpsAddr:           cfg.OpsAddr,
			},
			yacynodeassembly.LoadServiceConfig,
			yacynodeassembly.ListenersFor,
			yacynodeassembly.RunService,
		),
		memberAssemblyOf(
			yacydhtsearchMember,
			memberenvironment.Wiring{
				yacydhtsearchassembly.EnvNetworkName:   cfg.NetworkName,
				yacydhtsearchassembly.EnvSeedlistURLs:  cfg.SeedlistURLs,
				yacydhtsearchassembly.EnvServeProfiler: strconv.FormatBool(false),
				yacydhtsearchassembly.EnvOpsAddr:       cfg.OpsAddr,
			},
			yacydhtsearchassembly.LoadServiceConfig,
			yacydhtsearchassembly.ListenersFor,
			yacydhtsearchassembly.RunService,
		),
		memberAssemblyOf(
			corpustextMember,
			memberenvironment.Wiring{
				corpustextassembly.EnvPageOfferNATSURL: natsURL,
				corpustextassembly.EnvPageOfferDurable: corpustextMember,
				corpustextassembly.EnvBleveDirectory:   cfg.directoryOf(corpustextMember),
				corpustextassembly.EnvOpsAddr:          cfg.OpsAddr,
			},
			corpustextassembly.LoadServiceConfig,
			corpustextassembly.ListenersFor,
			corpustextassembly.RunService,
		),
		memberAssemblyOf(
			corpusmarkdownMember,
			memberenvironment.Wiring{
				corpusmarkdownassembly.EnvPageOfferNATSURL: natsURL,
				corpusmarkdownassembly.EnvPageOfferDurable: corpusmarkdownMember,
				corpusmarkdownassembly.EnvPageMarkdownDirectory: cfg.directoryOf(
					corpusmarkdownMember,
				),
				corpusmarkdownassembly.EnvOpsAddr: cfg.OpsAddr,
			},
			corpusmarkdownassembly.LoadServiceConfig,
			corpusmarkdownassembly.ListenersFor,
			corpusmarkdownassembly.RunService,
		),
	}
}

func (cfg ServiceConfig) directoryOf(memberName string) string {
	return filepath.Join(cfg.DataDirectory, memberName)
}

type serviceAssembly[C, L any] struct {
	wiring            memberenvironment.Wiring
	loadServiceConfig func(func(string) string) (C, error)
	listenersFor      func(context.Context, C) (L, error)
	runService        func(context.Context, C, L, *prometheus.Registry) error
	name              string
}

func memberAssemblyOf[C, L any](
	name string,
	wiring memberenvironment.Wiring,
	loadServiceConfig func(func(string) string) (C, error),
	listenersFor func(context.Context, C) (L, error),
	runService func(context.Context, C, L, *prometheus.Registry) error,
) memberAssembly {
	return serviceAssembly[C, L]{
		wiring:            wiring,
		loadServiceConfig: loadServiceConfig,
		listenersFor:      listenersFor,
		runService:        runService,
		name:              name,
	}
}

func (a serviceAssembly[C, L]) memberFor(
	ctx context.Context,
	getenv func(string) string,
) (member, error) {
	environment, err := memberenvironment.For(a.name, a.wiring, getenv)
	if err != nil {
		return member{}, err
	}
	cfg, err := a.loadServiceConfig(environment.Lookup)
	if err != nil {
		return member{}, fmt.Errorf("%s: %w", a.name, err)
	}
	listeners, err := a.listenersFor(ctx, cfg)
	if err != nil {
		return member{}, fmt.Errorf("%s: %w", a.name, err)
	}
	registry := opsmetrics.ProcessRegistry()
	return member{
		registry: registry,
		run: func(ctx context.Context) error {
			return a.runService(memberlog.ContextFor(ctx, a.name), cfg, listeners, registry)
		},
		name: a.name,
	}, nil
}
