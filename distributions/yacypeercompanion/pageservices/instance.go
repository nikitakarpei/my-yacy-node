// Package pageservices runs pagescrape and pagerwi for one paired peer in the companion app,
// and tells an observer when they connect, lose the broker, and end, and when the peer no longer
// accepts the device. Both connect to the broker of the peer with the device key, and pagescrape
// fetches each page through the capture proxy of the app.
package pageservices

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	pagerwiassembly "github.com/nikitakarpei/yacy-rwi-node/pagerwi/assembly"
	pagescrapeassembly "github.com/nikitakarpei/yacy-rwi-node/pagescrape/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/applog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servicehealth"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/brokeraccess"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/pairing"
)

const shutdownLimit = 15 * time.Second

type Observer interface {
	ServicesConnected()
	BrokerConnectionLost()
	ServicesStopped()
	BrokerUnreachable()
	ServicesFailed()
	DeviceUnpaired()
}

type Instance struct {
	membership       membershipTerms
	pagescrapeConfig pagescrapeassembly.ServiceConfig
	pagerwiConfig    pagerwiassembly.ServiceConfig
	stopped          chan struct{}
	stopOnce         sync.Once
}

func InstanceFor(
	linkText string,
	captureProxyURL string,
	key pairing.DeviceKey,
	appVersion string,
) (*Instance, error) {
	link, err := companionprotocol.PairLinkFrom(linkText)
	if err != nil {
		return nil, err
	}
	brokerOptions, err := brokeraccess.DeviceOptions(link, key)
	if err != nil {
		return nil, err
	}
	environment := environmentFor("tls://"+link.Addresses[0], captureProxyURL)
	lookup := func(name string) string { return environment[name] }
	if err := applog.Configure(lookup); err != nil {
		return nil, err
	}
	pagescrapeConfig, err := pagescrapeassembly.LoadServiceConfig(lookup)
	if err != nil {
		return nil, err
	}
	pagerwiConfig, err := pagerwiassembly.LoadServiceConfig(lookup)
	if err != nil {
		return nil, err
	}
	pagescrapeConfig.BrokerOptions = brokerOptions
	pagerwiConfig.BrokerOptions = brokerOptions
	return &Instance{
		membership: membershipTerms{
			link:          link,
			brokerOptions: brokerOptions,
			device:        companionprotocol.DeviceIDOf(key.PublicKeyInfo()),
			appVersion:    appVersion,
		},
		pagescrapeConfig: pagescrapeConfig,
		pagerwiConfig:    pagerwiConfig,
		stopped:          make(chan struct{}),
	}, nil
}

func environmentFor(brokerURL string, captureProxyURL string) map[string]string {
	return map[string]string{
		pagescrapeassembly.EnvScrapeNATSURL:    brokerURL,
		pagescrapeassembly.EnvProxyURL:         captureProxyURL,
		pagescrapeassembly.EnvProxyDialMode:    pagefetchershttp.ProxyDialAbsoluteURLName,
		pagescrapeassembly.EnvProvisionStreams: strconv.FormatBool(false),
		pagerwiassembly.EnvPageOfferNATSURL:    brokerURL,
		pagerwiassembly.EnvPageRWINATSURL:      brokerURL,
		pagerwiassembly.EnvProvisionStreams:    strconv.FormatBool(false),
	}
}

func (i *Instance) Run(observer Observer) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-i.stopped:
			cancel()
		case <-ctx.Done():
		}
	}()
	var unpaired atomic.Bool
	member, err := i.membership.joined(func() {
		unpaired.Store(true)
		cancel()
	})
	if err != nil {
		reportRefusedJoin(err, observer)
		return
	}
	defer member.leave()
	pagescrapeConfig := i.pagescrapeConfig
	pagescrapeConfig.ScrapeNATSURL = member.brokerURL()
	pagerwiConfig := i.pagerwiConfig
	pagerwiConfig.PageOfferNATSURL = member.brokerURL()
	pagerwiConfig.PageRWINATSURL = member.brokerURL()
	pagerwiConfig.PageRWIPartitionExponent = member.settings.PartitionExponent
	connection := newConnectionWatch(observer)
	err = servergroup.Run(
		ctx,
		shutdownLimit,
		nil,
		func(ctx context.Context) error {
			return runPagescrape(ctx, pagescrapeConfig, connection.pagescrapeHealth)
		},
		func(ctx context.Context) error {
			return runPagerwi(ctx, pagerwiConfig, connection.pagerwiHealth)
		},
	)
	connection.end()
	if unpaired.Load() || err != nil && i.membership.refused() {
		observer.DeviceUnpaired()
		return
	}
	reportEnding(err, observer)
}

func reportRefusedJoin(err error, observer Observer) {
	slog.WarnContext(context.Background(), "peer not joined", slog.Any("error", err))
	if errors.Is(err, nats.ErrAuthorization) {
		observer.DeviceUnpaired()
		return
	}
	observer.BrokerUnreachable()
}

func runPagescrape(
	ctx context.Context,
	cfg pagescrapeassembly.ServiceConfig,
	health *servicehealth.Health,
) error {
	listeners, err := pagescrapeassembly.ListenersFor(ctx, cfg)
	if err != nil {
		return err
	}
	return pagescrapeassembly.RunService(
		ctx,
		cfg,
		listeners,
		(&net.Dialer{}).DialContext,
		prometheus.NewRegistry(),
		health,
	)
}

func runPagerwi(
	ctx context.Context,
	cfg pagerwiassembly.ServiceConfig,
	health *servicehealth.Health,
) error {
	listeners, err := pagerwiassembly.ListenersFor(ctx, cfg)
	if err != nil {
		return err
	}
	return pagerwiassembly.RunService(
		ctx,
		cfg,
		listeners,
		(&net.Dialer{}).DialContext,
		prometheus.NewRegistry(),
		health,
	)
}

func reportEnding(err error, observer Observer) {
	if err != nil {
		slog.WarnContext(context.Background(), "page services ended", slog.Any("error", err))
	}
	switch {
	case err == nil:
		observer.ServicesStopped()
	case errors.Is(err, jetstreamconnect.ErrBrokerUnreachable),
		errors.Is(err, pullintake.ErrBrokerGone):
		observer.BrokerUnreachable()
	default:
		observer.ServicesFailed()
	}
}

func (i *Instance) Stop() {
	i.stopOnce.Do(func() { close(i.stopped) })
}
