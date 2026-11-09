package yacypeerassembly

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/brokerkey"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionaccess"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionpresence"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionrequests"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionsettings"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionspage"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/devicepermissions"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/paireddevices"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/pairingoffers"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/peerintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/processnetwork"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/thishost"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/webuitabs"
)

const (
	companionsDirectoryName = "companions"
	pairingSecretLifetime   = time.Minute
	pairingWait             = 30 * time.Second
)

type companions interface {
	startBroker(directory string) (*natsbroker.Broker, error)
	IntakeAdmission() pullintake.Admission
	runners(
		broker *natsbroker.Broker,
		natsURL string,
		network *processnetwork.Network,
	) []func(context.Context) error
	tabLinks() webuitabs.Links
	register(mux *http.ServeMux, broker *natsbroker.Broker, tabBar webuistyle.TabBar)
	close()
}

type noCompanions struct{}

type servedCompanions struct {
	devices           *paireddevices.Devices
	offers            *pairingoffers.Offers
	settings          *companionsettings.Keeper
	intakeAdmission   *peerintake.Admission
	certificate       tls.Certificate
	address           string
	partitionExponent uint
}

func openCompanions(ctx context.Context, cfg PeerConfig) (companions, error) {
	if !cfg.servesCompanions() {
		return noCompanions{}, nil
	}
	directory := cfg.directoryOf(companionsDirectoryName)
	certificate, err := brokerkey.Settle(directory)
	if err != nil {
		return nil, err
	}
	intakeAdmission := peerintake.New()
	devices, err := paireddevices.Open(ctx, directory, intakeAdmission)
	if err != nil {
		return nil, err
	}
	settings, err := companionsettings.Open(ctx, directory, intakeAdmission)
	if err != nil {
		devices.Close()
		return nil, err
	}
	return &servedCompanions{
		devices:           devices,
		offers:            pairingoffers.New(wallclock.Clock{}, pairingSecretLifetime, pairingWait),
		settings:          settings,
		intakeAdmission:   intakeAdmission,
		certificate:       certificate,
		address:           cfg.CompanionAddr,
		partitionExponent: cfg.PartitionExponent,
	}, nil
}

func (noCompanions) startBroker(directory string) (*natsbroker.Broker, error) {
	return natsbroker.Start(directory)
}

func (noCompanions) IntakeAdmission() pullintake.Admission {
	return pullintake.EveryItemAdmitted{}
}

func (noCompanions) runners(
	*natsbroker.Broker,
	string,
	*processnetwork.Network,
) []func(context.Context) error {
	return nil
}

func (noCompanions) tabLinks() webuitabs.Links {
	return nil
}

func (noCompanions) register(*http.ServeMux, *natsbroker.Broker, webuistyle.TabBar) {}

func (noCompanions) close() {}

func (c *servedCompanions) startBroker(directory string) (*natsbroker.Broker, error) {
	return natsbroker.StartWithEndpoint(directory, c.endpoint())
}

func (c *servedCompanions) endpoint() natsbroker.Endpoint {
	return natsbroker.Endpoint{
		Authentication: companionaccess.New(
			c.devices,
			c.offers,
			wallclock.Clock{},
			devicepermissions.ForPageServices(),
			pairingWait,
		),
		Address:     c.address,
		Certificate: c.certificate,
	}
}

func (c *servedCompanions) IntakeAdmission() pullintake.Admission {
	return c.intakeAdmission
}

func (c *servedCompanions) brokerKeyHash() companionprotocol.BrokerKeyHash {
	return companionprotocol.BrokerKeyHashFrom(c.certificate.Leaf.RawSubjectPublicKeyInfo)
}

func (c *servedCompanions) tabLinks() webuitabs.Links {
	return webuitabs.Links{{Title: webuitabs.Companions, URL: companionspage.Path}}
}

func (c *servedCompanions) register(
	mux *http.ServeMux,
	broker *natsbroker.Broker,
	tabBar webuistyle.TabBar,
) {
	companionspage.Register(mux, companionspage.Collaborators{
		Devices:           c.devices,
		Unpairer:          companionaccess.NewUnpairer(c.devices, broker),
		Offers:            offerOpener{offers: c.offers},
		DeviceConnections: companionaccess.NewDeviceConnections(broker),
		Settings:          c.settings,
		FetchingPause:     c.intakeAdmission,
	}, c.pairLinkFor(broker), tabBar)
}

func (c *servedCompanions) pairLinkFor(broker *natsbroker.Broker) companionprotocol.PairLink {
	address := broker.EndpointAddress().(*net.TCPAddr) //nolint:forcetypeassert // the endpoint listens on TCP.
	return companionprotocol.PairLink{
		Addresses:     thishost.Host().AddressesFor(address),
		BrokerKeyHash: c.brokerKeyHash(),
	}
}

func (c *servedCompanions) runners(
	broker *natsbroker.Broker,
	natsURL string,
	network *processnetwork.Network,
) []func(context.Context) error {
	pairingAnswerer := companionrequests.NewPairingAnswerer(
		c.offers, c.devices, wallclock.Clock{}, c.brokerKeyHash(),
	)
	deviceAnswerer := companionrequests.NewDeviceAnswerer(
		c.devices,
		companionaccess.NewUnpairer(c.devices, broker),
		companionprotocol.DeviceSettings{
			PartitionExponent: c.partitionExponent,
		},
	)
	return []func(context.Context) error{
		runnerFor(pairingAnswerer, natsURL, network),
		runnerFor(deviceAnswerer, natsURL, network),
		companionpresence.New(
			companionaccess.NewDeviceConnections(broker), wallclock.Clock{}, c.intakeAdmission,
		).Run,
	}
}

type answerer interface {
	Subscribe(ctx context.Context, connection *nats.Conn) (*companionrequests.Subscriptions, error)
}

func runnerFor(
	answerer answerer,
	natsURL string,
	network *processnetwork.Network,
) func(context.Context) error {
	return func(ctx context.Context) error {
		_, connection, err := jetstreamconnect.Open(
			natsURL,
			network.DialContext,
			jetstreamconnect.UnobservedConnection{},
			jetstreamconnect.BrokerAccess{},
		)
		if err != nil {
			return err
		}
		defer connection.Close()
		subscriptions, err := answerer.Subscribe(ctx, connection)
		if err != nil {
			return err
		}
		subscriptions.EndWhenDone(ctx)
		return nil
	}
}

func (c *servedCompanions) close() {
	c.devices.Close()
	c.settings.Close()
}

type offerOpener struct {
	offers *pairingoffers.Offers
}

func (o offerOpener) Open() companionspage.PairingOffer {
	return o.offers.Open()
}
