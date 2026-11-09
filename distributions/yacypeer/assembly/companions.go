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
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/brokerkey"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionaccess"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionrequests"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionspage"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/devicepermissions"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/paireddevices"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/pairingoffers"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/processnetwork"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/thishost"
)

const (
	companionsDirectoryName = "companions"
	pairingSecretLifetime   = time.Minute
	pairingWait             = 30 * time.Second
)

type companions struct {
	devices           *paireddevices.Devices
	offers            *pairingoffers.Offers
	certificate       tls.Certificate
	address           string
	partitionExponent uint
}

func openCompanions(cfg PeerConfig) (*companions, error) {
	directory := cfg.directoryOf(companionsDirectoryName)
	certificate, err := brokerkey.Settle(directory)
	if err != nil {
		return nil, err
	}
	devices, err := paireddevices.Open(directory)
	if err != nil {
		return nil, err
	}
	return &companions{
		devices:           devices,
		offers:            pairingoffers.New(wallclock.Clock{}, pairingSecretLifetime, pairingWait),
		certificate:       certificate,
		address:           cfg.CompanionAddr,
		partitionExponent: cfg.PartitionExponent,
	}, nil
}

func (c *companions) endpoint() natsbroker.Endpoint {
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

func (c *companions) brokerKeyHash() companionprotocol.BrokerKeyHash {
	return companionprotocol.BrokerKeyHashFrom(c.certificate.Leaf.RawSubjectPublicKeyInfo)
}

func (c *companions) register(
	mux *http.ServeMux,
	broker *natsbroker.Broker,
	tabBar webuistyle.TabBar,
) {
	companionspage.Register(mux, companionspage.Collaborators{
		Devices:           c.devices,
		Unpairer:          companionaccess.NewUnpairer(c.devices, broker),
		Offers:            offerOpener{offers: c.offers},
		DeviceConnections: companionaccess.NewDeviceConnections(broker),
	}, c.pairLinkFor(broker), tabBar)
}

func (c *companions) pairLinkFor(broker *natsbroker.Broker) companionprotocol.PairLink {
	address := broker.EndpointAddress().(*net.TCPAddr) //nolint:forcetypeassert // the endpoint listens on TCP.
	return companionprotocol.PairLink{
		Addresses:     thishost.Host().AddressesFor(address),
		BrokerKeyHash: c.brokerKeyHash(),
	}
}

func (c *companions) runners(
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

func (c *companions) close() {
	c.devices.Close()
}

type offerOpener struct {
	offers *pairingoffers.Offers
}

func (o offerOpener) Open() companionspage.PairingOffer {
	return o.offers.Open()
}
