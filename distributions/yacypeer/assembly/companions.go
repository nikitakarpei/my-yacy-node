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
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/brokerkey"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionaccess"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionpresence"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionrequests"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionsettings"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionspage"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/paireddevices"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/pairingoffers"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/peerintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/hostaddresses"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/processnetwork"
)

const (
	companionsDirectoryName = "companions"
	pairingSecretLifetime   = time.Minute
	pairingWait             = 30 * time.Second
)

type companions interface {
	startBroker(directory string) (*natsbroker.Broker, error)
	intakeAdmission() pullintake.Admission
	register(mux *http.ServeMux, broker *natsbroker.Broker, tabBar webuistyle.TabBar)
	servers(
		broker *natsbroker.Broker,
		natsURL string,
		network *processnetwork.Network,
	) []func(context.Context) error
	close(ctx context.Context)
}

type noCompanions struct{}

type pairedCompanions struct {
	devices     *paireddevices.Devices
	offers      *pairingoffers.Offers
	settings    *companionsettings.Settings
	intakeHold  *peerintake.IntakeHold
	certificate tls.Certificate
	address     string
}

func openCompanions(ctx context.Context, cfg PeerConfig) (companions, error) {
	if !cfg.pairsCompanions() {
		return noCompanions{}, nil
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
	settings, err := companionsettings.Open(ctx, directory, intakeHold)
	if err != nil {
		return nil, errors.Join(err, devices.Close())
	}
	return &pairedCompanions{
		devices:     devices,
		offers:      pairingoffers.New(wallclock.Clock{}, pairingSecretLifetime, pairingWait),
		settings:    settings,
		intakeHold:  intakeHold,
		certificate: certificate,
		address:     cfg.CompanionAddr,
	}, nil
}

func (noCompanions) startBroker(directory string) (*natsbroker.Broker, error) {
	return natsbroker.Start(directory)
}

func (noCompanions) intakeAdmission() pullintake.Admission {
	return nil
}

func (noCompanions) register(*http.ServeMux, *natsbroker.Broker, webuistyle.TabBar) {}

func (noCompanions) servers(
	*natsbroker.Broker,
	string,
	*processnetwork.Network,
) []func(context.Context) error {
	return nil
}

func (noCompanions) close(context.Context) {}

func (c *pairedCompanions) startBroker(directory string) (*natsbroker.Broker, error) {
	return natsbroker.StartWithEndpoint(directory, c.endpoint())
}

func (c *pairedCompanions) endpoint() natsbroker.Endpoint {
	return natsbroker.Endpoint{
		Authentication: companionaccess.New(
			c.devices,
			c.offers,
			wallclock.Clock{},
			pageServicesPermissions(),
			pairingWait,
		),
		Address:     c.address,
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

func (c *pairedCompanions) intakeAdmission() pullintake.Admission {
	return c.intakeHold
}

func (c *pairedCompanions) register(
	mux *http.ServeMux,
	broker *natsbroker.Broker,
	tabBar webuistyle.TabBar,
) {
	companionspage.Register(mux, companionspage.Collaborators{
		Devices:     c.devices,
		Unpairer:    c.unpairerFor(broker),
		Offers:      offerOpener{offers: c.offers},
		Connections: companionaccess.NewDeviceConnections(broker),
		Settings:    c.settings,
	}, c.pairLinkFor(broker), tabBar)
}

func (c *pairedCompanions) unpairerFor(broker *natsbroker.Broker) companionaccess.Unpairer {
	return companionaccess.NewUnpairer(c.devices, broker)
}

func (c *pairedCompanions) pairLinkFor(broker *natsbroker.Broker) companionprotocol.PairLink {
	address := broker.EndpointAddress().(*net.TCPAddr) //nolint:forcetypeassert // the endpoint listens on TCP.
	return companionprotocol.PairLink{
		Addresses:     hostaddresses.ThisHost().AddressesFor(address),
		BrokerKeyHash: c.brokerKeyHash(),
	}
}

func (c *pairedCompanions) brokerKeyHash() string {
	return companionprotocol.BrokerKeyHashOf(c.certificate.Leaf.RawSubjectPublicKeyInfo)
}

func (c *pairedCompanions) servers(
	broker *natsbroker.Broker,
	natsURL string,
	network *processnetwork.Network,
) []func(context.Context) error {
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
		companionpresence.New(
			companionaccess.NewDeviceConnections(broker), wallclock.Clock{}, c.intakeHold,
		).Run,
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

func (c *pairedCompanions) close(ctx context.Context) {
	if err := errors.Join(c.devices.Close(), c.settings.Close()); err != nil {
		slog.WarnContext(ctx, "companion files did not close", slog.Any("error", err))
	}
}

type offerOpener struct {
	offers *pairingoffers.Offers
}

func (o offerOpener) Open() companionspage.PairingOffer {
	return o.offers.Open()
}
