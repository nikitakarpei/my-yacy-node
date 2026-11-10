// Package pageservices runs pagescrape and pagerwi for one paired peer in the companion app,
// and tells an observer when they connect, lose the broker, and end, when the peer no longer
// accepts the device, and when the app or the peer is outdated for the other. Both connect to the broker of the peer with the device key, and pagescrape
// fetches each page through the capture proxy of the app. Both sign their statements with the
// hardware keys of the app.
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
	"github.com/nikitakarpei/hardware-attested-statements/verifier/profiles/android"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/attestedstatements"
	attestedstatementsobserversapplog "github.com/nikitakarpei/yacy-rwi-node/attestedstatements/attestedstatementsobservers/applog"
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
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/pairingrequests"
)

const shutdownLimit = 15 * time.Second

type Observer interface {
	ServicesConnected()
	BrokerConnectionLost()
	ServicesStopped()
	BrokerUnreachable()
	ServicesFailed()
	DeviceRefused()
	AppOutdated()
	PeerOutdated()
}

type Instance struct {
	terms           sessionTerms
	captureProxyURL string
	statementSigner attestedstatements.Signer
	stopped         chan struct{}
	stopOnce        sync.Once
}

func InstanceFor(
	linkText string,
	captureProxyURL string,
	key pairingrequests.DeviceKey,
	appVersion string,
	hardwareKeys HardwareKeys,
) (*Instance, error) {
	link, err := companionprotocol.PairLinkFrom(linkText)
	if err != nil {
		return nil, err
	}
	brokerAccess, err := brokeraccess.DeviceAccess(link, key)
	if err != nil {
		return nil, err
	}
	if err := applog.Configure(func(string) string { return "" }); err != nil {
		return nil, err
	}
	return &Instance{
		terms: sessionTerms{
			link:         link,
			brokerAccess: brokerAccess,
			deviceID:     companionprotocol.DeviceIDFrom(key.PublicKeyInfo()),
			appVersion:   appVersion,
		},
		captureProxyURL: captureProxyURL,
		statementSigner: attestedstatements.New(
			statementKeys{hardwareKeys: hardwareKeys},
			android.New(nil),
			attestedstatementsobserversapplog.SigningLog{},
		),
		stopped: make(chan struct{}),
	}, nil
}

func (i *Instance) Run(observer Observer) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go i.cancelOnStop(ctx, cancel)
	var deviceRefused atomic.Bool
	session, err := i.terms.openSession(func() {
		deviceRefused.Store(true)
		cancel()
	})
	if err != nil {
		reportUnopenedSession(err, observer)
		return
	}
	defer session.close()
	err = i.runPageServicesOver(ctx, session, observer)
	if deviceRefused.Load() || err != nil && i.terms.probeRefusal() {
		reportRefusal(err, observer)
		return
	}
	reportEnding(err, observer)
}

func (i *Instance) cancelOnStop(ctx context.Context, cancel context.CancelFunc) {
	select {
	case <-i.stopped:
		cancel()
	case <-ctx.Done():
	}
}

func reportUnopenedSession(err error, observer Observer) {
	slog.WarnContext(context.Background(), "peer session not opened", slog.Any("error", err))
	switch {
	case errors.Is(err, nats.ErrAuthorization):
		observer.DeviceRefused()
	case errors.Is(err, brokeraccess.ErrNoConnection):
		observer.BrokerUnreachable()
	case errors.Is(err, errAppOutdated):
		observer.AppOutdated()
	case errors.Is(err, errPeerOutdated):
		observer.PeerOutdated()
	default:
		observer.ServicesFailed()
	}
}

func (i *Instance) runPageServicesOver(
	ctx context.Context,
	session *peerSession,
	observer Observer,
) error {
	configs, err := pageServiceConfigsFor(
		session.brokerURL(), i.captureProxyURL, session.partitionExponent(),
	)
	if err != nil {
		return err
	}
	connection := newConnectionWatch(observer)
	err = servergroup.Run(
		ctx,
		shutdownLimit,
		nil,
		func(ctx context.Context) error {
			return runPagescrape(
				ctx,
				configs.pagescrape,
				i.terms.brokerAccess,
				i.statementSigner,
				connection.pagescrapeHealth,
			)
		},
		func(ctx context.Context) error {
			return runPagerwi(
				ctx,
				configs.pagerwi,
				i.terms.brokerAccess,
				i.statementSigner,
				connection.pagerwiHealth,
			)
		},
	)
	connection.end()
	return err
}

type pageServiceConfigs struct {
	pagescrape pagescrapeassembly.ServiceConfig
	pagerwi    pagerwiassembly.ServiceConfig
}

func pageServiceConfigsFor(
	brokerURL string,
	captureProxyURL string,
	partitionExponent uint,
) (pageServiceConfigs, error) {
	environment := environmentFor(brokerURL, captureProxyURL, partitionExponent)
	lookup := func(name string) string { return environment[name] }
	pagescrapeConfig, err := pagescrapeassembly.LoadServiceConfig(lookup)
	if err != nil {
		return pageServiceConfigs{}, err
	}
	pagerwiConfig, err := pagerwiassembly.LoadServiceConfig(lookup)
	if err != nil {
		return pageServiceConfigs{}, err
	}
	return pageServiceConfigs{pagescrape: pagescrapeConfig, pagerwi: pagerwiConfig}, nil
}

func environmentFor(
	brokerURL string,
	captureProxyURL string,
	partitionExponent uint,
) map[string]string {
	return map[string]string{
		pagescrapeassembly.EnvScrapeNATSURL:    brokerURL,
		pagescrapeassembly.EnvProxyURL:         captureProxyURL,
		pagescrapeassembly.EnvProxyDialMode:    pagefetchershttp.ProxyDialAbsoluteURLName,
		pagescrapeassembly.EnvProvisionStreams: strconv.FormatBool(false),
		pagerwiassembly.EnvPageOfferNATSURL:    brokerURL,
		pagerwiassembly.EnvPageRWINATSURL:      brokerURL,
		pagerwiassembly.EnvPageRWIPartitionExponent: strconv.FormatUint(
			uint64(partitionExponent),
			10,
		),
		pagerwiassembly.EnvProvisionStreams: strconv.FormatBool(false),
	}
}

func runPagescrape(
	ctx context.Context,
	cfg pagescrapeassembly.ServiceConfig,
	brokerAccess jetstreamconnect.BrokerAccess,
	statementSigner attestedstatements.Signer,
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
		brokerAccess,
		pullintake.EveryItemAdmitted{},
		statementSigner,
		prometheus.NewRegistry(),
		health,
	)
}

func runPagerwi(
	ctx context.Context,
	cfg pagerwiassembly.ServiceConfig,
	brokerAccess jetstreamconnect.BrokerAccess,
	statementSigner attestedstatements.Signer,
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
		brokerAccess,
		brokerAccess,
		pullintake.EveryItemAdmitted{},
		statementSigner,
		prometheus.NewRegistry(),
		health,
	)
}

func reportRefusal(err error, observer Observer) {
	if err != nil {
		slog.WarnContext(context.Background(), "page services ended as the peer refused the device",
			slog.Any("error", err),
		)
	}
	observer.DeviceRefused()
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
