package yacypeerassembly

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	corpusmarkdownassembly "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/assembly"
	corpustextassembly "github.com/nikitakarpei/yacy-rwi-node/corpustext/assembly"
	yacydhtsearchassembly "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/assembly"
	yacynodeassembly "github.com/nikitakarpei/yacy-rwi-node/yacynode/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/memberenvironment"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/memberlog"
)

var unreadSettings = []string{
	yacynodeassembly.EnvOpsAddr,
	yacydhtsearchassembly.EnvOpsAddr,
	yacydhtsearchassembly.EnvServeProfiler,
	corpustextassembly.EnvOpsAddr,
	corpusmarkdownassembly.EnvOpsAddr,
}

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

type memberAssembly func(ctx context.Context, getenv func(string) string) (member, error)

func membersFor(
	ctx context.Context,
	cfg ServiceConfig,
	natsURL string,
	getenv func(string) string,
) ([]member, error) {
	assemblies := memberAssembliesFor(cfg, natsURL)
	members := make([]member, 0, len(assemblies))
	for _, assembly := range assemblies {
		member, err := assembly(ctx, getenv)
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
			},
			yacynodeassembly.LoadServiceConfig,
			yacynodeassembly.ListenersFor,
			yacynodeassembly.RunService,
		),
		memberAssemblyOf(
			yacydhtsearchMember,
			memberenvironment.Wiring{
				yacydhtsearchassembly.EnvNetworkName:  cfg.NetworkName,
				yacydhtsearchassembly.EnvSeedlistURLs: cfg.SeedlistURLs,
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

func memberAssemblyOf[C, L any](
	name string,
	wiring memberenvironment.Wiring,
	loadServiceConfig func(func(string) string) (C, error),
	listenersFor func(context.Context, C) (L, error),
	runService func(context.Context, C, L, *prometheus.Registry) error,
) memberAssembly {
	return func(ctx context.Context, getenv func(string) string) (member, error) {
		environment, err := memberenvironment.For(name, wiring, unreadSettings, getenv)
		if err != nil {
			return member{}, err
		}
		cfg, err := loadServiceConfig(environment.Lookup)
		if err != nil {
			return member{}, fmt.Errorf("%s: %w", name, err)
		}
		listeners, err := listenersFor(ctx, cfg)
		if err != nil {
			return member{}, fmt.Errorf("%s: %w", name, err)
		}
		registry := prometheus.NewRegistry()
		return member{
			registry: registry,
			run: func(ctx context.Context) error {
				return runService(memberlog.ContextFor(ctx, name), cfg, listeners, registry)
			},
			name: name,
		}, nil
	}
}
