package yacypeerassembly

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	pagerwiassembly "github.com/nikitakarpei/yacy-rwi-node/pagerwi/assembly"
	pagescrapeassembly "github.com/nikitakarpei/yacy-rwi-node/pagescrape/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/natspermissions"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companionpresence"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/brokerkey"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionaccess"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionrequests"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionspage"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/paireddevices"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/pairingoffers"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/fetchsettings"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/hostaddresses"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/peerintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/processnetwork"
)

const (
	companionsDirectoryName = "companions"
	pairingSecretLifetime   = time.Minute
	pairingWait             = 30 * time.Second
)

type companions struct {
	devices     *paireddevices.Devices
	offers      *pairingoffers.Offers
	settings    *fetchsettings.Settings
	intakeHold  *peerintake.IntakeHold
	certificate tls.Certificate
}

func openCompanions(ctx context.Context, cfg PeerConfig) (*companions, error) {
	if !cfg.pairsCompanions() {
		return nil, nil
	}
	directory := cfg.directoryOf(companionsDirectoryName)
	certificate, err := brokerkey.Settle(directory)
	if err != nil {
		return nil, err
	}
	intakeHold := peerintake.NewIntakeHold()
	devices, err := paireddevices.Open(ctx, directory, intakeHold)
	if err != nil {
		return nil, err
	}
	settings, err := fetchsettings.Open(ctx, directory, intakeHold)
	if err != nil {
		return nil, errors.Join(err, devices.Close())
	}
	return &companions{
		devices:     devices,
		offers:      pairingoffers.New(wallclock.Clock{}, pairingSecretLifetime, pairingWait),
		settings:    settings,
		intakeHold:  intakeHold,
		certificate: certificate,
	}, nil
}

func startBroker(cfg PeerConfig, parts *companions) (*natsbroker.Broker, error) {
	directory := cfg.directoryOf(brokerDirectoryName)
	if parts == nil {
		return natsbroker.Start(directory)
	}
	return natsbroker.StartWithEndpoint(directory, parts.endpointAt(cfg.CompanionAddr))
}

func (c *companions) endpointAt(address string) natsbroker.Endpoint {
	return natsbroker.Endpoint{
		Authentication: companionaccess.New(
			c.devices,
			c.offers,
			wallclock.Clock{},
			pageServicesPermissions(),
			pairingWait,
		),
		Address:     address,
		Certificate: c.certificate,
	}
}

func pageServicesPermissions() natspermissions.Permissions {
	return natspermissions.JoinedFrom(
		pagescrapeassembly.ServiceConfig{
			ScrapeRequestDurable: pagescrapeassembly.DefaultScrapeRequestDurable,
		}.BrokerPermissions(),
		pagerwiassembly.ServiceConfig{
			PageOfferDurable: pagerwiassembly.DefaultPageOfferDurable,
		}.BrokerPermissions(),
	)
}

func (c *companions) close() error {
	return errors.Join(c.devices.Close(), c.settings.Close())
}

func (c *companions) unpairerFor(broker *natsbroker.Broker) companionaccess.Unpairer {
	return companionaccess.NewUnpairer(c.devices, broker)
}

func (c *companions) intakeAdmission() pullintake.Admission {
	if c == nil {
		return nil
	}
	return c.intakeHold
}

func (c *companions) register(
	mux *http.ServeMux,
	broker *natsbroker.Broker,
	tabBar webuistyle.TabBar,
) {
	if c == nil {
		return
	}
	companionspage.Register(mux, companionspage.Collaborators{
		Devices:  c.devices,
		Unpairer: c.unpairerFor(broker),
		Offers:   offerOpener{offers: c.offers},
		Broker:   broker,
		Settings: c.settings,
	}, c.pairLinkFor(broker), tabBar)
}

func (c *companions) pairLinkFor(broker *natsbroker.Broker) companionprotocol.PairLink {
	address := broker.EndpointAddress().(*net.TCPAddr) //nolint:forcetypeassert // the endpoint listens on TCP.
	return companionprotocol.PairLink{
		Addresses:     hostaddresses.ThisHost().AddressesFor(address),
		BrokerKeyHash: c.brokerKeyHash(),
	}
}

func (c *companions) brokerKeyHash() string {
	return companionprotocol.BrokerKeyHashOf(c.certificate.Leaf.RawSubjectPublicKeyInfo)
}

func (c *companions) servers(
	broker *natsbroker.Broker,
	natsURL string,
	network *processnetwork.Network,
) []func(context.Context) error {
	if c == nil {
		return nil
	}
	pairingRequests := companionrequests.NewPairingRequests(
		c.offers, c.devices, wallclock.Clock{}, c.brokerKeyHash(),
	)
	deviceRequests := companionrequests.NewDeviceRequests(
		c.devices,
		c.unpairerFor(broker),
		companionprotocol.Settings{
			PartitionExponent: pagerwiassembly.DefaultPageRWIPartitionExponent,
		},
	)
	return []func(context.Context) error{
		servedOn(natsURL, network, pairingRequests.Serve),
		servedOn(natsURL, network, deviceRequests.Serve),
		companionpresence.New(broker, wallclock.Clock{}, c.intakeHold).Run,
	}
}

func servedOn(
	natsURL string,
	network *processnetwork.Network,
	serve func(context.Context, *nats.Conn) error,
) func(context.Context) error {
	return func(ctx context.Context) error {
		_, connection, err := jetstreamconnect.Open(natsURL, network.DialContext)
		if err != nil {
			return err
		}
		defer connection.Close()
		return serve(ctx, connection)
	}
}

func closeCompanions(ctx context.Context, parts *companions) {
	if parts == nil {
		return
	}
	if err := parts.close(); err != nil {
		slog.WarnContext(ctx, "companion files did not close", slog.Any("error", err))
	}
}

type offerOpener struct {
	offers *pairingoffers.Offers
}

func (o offerOpener) Open() companionspage.PairingOffer {
	return o.offers.Open()
}
