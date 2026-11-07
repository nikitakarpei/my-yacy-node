// Package pageservices runs pagescrape and pagerwi for one yacypeer in the companion app,
// and tells an observer when they connect, lose the broker, and end. Both use the broker
// that the connect link names, and pagescrape fetches each page through the capture proxy
// of the app.
package pageservices

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	pagerwiassembly "github.com/nikitakarpei/yacy-rwi-node/pagerwi/assembly"
	pagescrapeassembly "github.com/nikitakarpei/yacy-rwi-node/pagescrape/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/applog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servicehealth"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/connectlink"
)

const shutdownLimit = 15 * time.Second

type Observer interface {
	ServicesConnected()
	BrokerConnectionLost()
	ServicesStopped()
	BrokerUnreachable()
	ServicesFailed()
}

type Instance struct {
	pagescrapeConfig pagescrapeassembly.ServiceConfig
	pagerwiConfig    pagerwiassembly.ServiceConfig
	stopped          chan struct{}
	stopOnce         sync.Once
}

func InstanceFor(linkText string, captureProxyURL string) (*Instance, error) {
	link, err := connectlink.LinkFrom(linkText)
	if err != nil {
		return nil, err
	}
	environment := environmentFor(link, captureProxyURL)
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
	return &Instance{
		pagescrapeConfig: pagescrapeConfig,
		pagerwiConfig:    pagerwiConfig,
		stopped:          make(chan struct{}),
	}, nil
}

func environmentFor(link *connectlink.Link, captureProxyURL string) map[string]string {
	return map[string]string{
		pagescrapeassembly.EnvScrapeNATSURL:         link.NATSURL,
		pagescrapeassembly.EnvProxyURL:              captureProxyURL,
		pagescrapeassembly.EnvProxyDialMode:         pagefetchershttp.ProxyDialAbsoluteURLName,
		pagerwiassembly.EnvPageOfferNATSURL:         link.NATSURL,
		pagerwiassembly.EnvPageRWINATSURL:           link.NATSURL,
		pagerwiassembly.EnvPageRWIPartitionExponent: strconv.Itoa(link.PartitionExponent),
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
	connection := newConnectionWatch(observer)
	err := servergroup.Run(
		ctx,
		shutdownLimit,
		nil,
		func(ctx context.Context) error { return i.runPagescrape(ctx, connection.pagescrapeHealth) },
		func(ctx context.Context) error { return i.runPagerwi(ctx, connection.pagerwiHealth) },
	)
	connection.end()
	reportEnding(err, observer)
}

func (i *Instance) runPagescrape(ctx context.Context, health *servicehealth.Health) error {
	listeners, err := pagescrapeassembly.ListenersFor(ctx, i.pagescrapeConfig)
	if err != nil {
		return err
	}
	return pagescrapeassembly.RunService(
		ctx,
		i.pagescrapeConfig,
		listeners,
		(&net.Dialer{}).DialContext,
		prometheus.NewRegistry(),
		health,
	)
}

func (i *Instance) runPagerwi(ctx context.Context, health *servicehealth.Health) error {
	listeners, err := pagerwiassembly.ListenersFor(ctx, i.pagerwiConfig)
	if err != nil {
		return err
	}
	return pagerwiassembly.RunService(
		ctx,
		i.pagerwiConfig,
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
