package yacypeerassembly

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	pagerwiassembly "github.com/nikitakarpei/yacy-rwi-node/pagerwi/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/brokerkey"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionaccess"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionrequests"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/devicepermissions"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/paireddevices"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/pairingoffers"
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
	servers(
		broker *natsbroker.Broker,
		natsURL string,
		network *processnetwork.Network,
	) []func(context.Context) error
	close()
}

type noCompanions struct{}

type servedCompanions struct {
	devices     *paireddevices.Devices
	offers      *pairingoffers.Offers
	certificate tls.Certificate
	address     string
}

func openCompanions(cfg PeerConfig) (companions, error) {
	if !cfg.servesCompanions() {
		return noCompanions{}, nil
	}
	directory := cfg.directoryOf(companionsDirectoryName)
	certificate, err := brokerkey.Settle(directory)
	if err != nil {
		return nil, err
	}
	devices, err := paireddevices.Open(directory)
	if err != nil {
		return nil, err
	}
	return &servedCompanions{
		devices:     devices,
		offers:      pairingoffers.New(wallclock.Clock{}, pairingSecretLifetime, pairingWait),
		certificate: certificate,
		address:     cfg.CompanionAddr,
	}, nil
}

func (noCompanions) startBroker(directory string) (*natsbroker.Broker, error) {
	return natsbroker.Start(directory)
}

func (noCompanions) servers(
	*natsbroker.Broker,
	string,
	*processnetwork.Network,
) []func(context.Context) error {
	return nil
}

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

func (c *servedCompanions) brokerKeyHash() companionprotocol.BrokerKeyHash {
	return companionprotocol.BrokerKeyHashFrom(c.certificate.Leaf.RawSubjectPublicKeyInfo)
}

func (c *servedCompanions) servers(
	broker *natsbroker.Broker,
	natsURL string,
	network *processnetwork.Network,
) []func(context.Context) error {
	pairingRequests := companionrequests.NewPairingAnswerer(
		c.offers, c.devices, wallclock.Clock{}, c.brokerKeyHash(),
	)
	deviceRequests := companionrequests.NewDeviceAnswerer(
		c.devices,
		companionaccess.NewUnpairer(c.devices, broker),
		companionprotocol.DeviceSettings{
			PartitionExponent: pagerwiassembly.DefaultPageRWIPartitionExponent,
		},
	)
	return []func(context.Context) error{
		servedOn(natsURL, network, pairingRequests.Serve),
		servedOn(natsURL, network, deviceRequests.Serve),
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

func (c *servedCompanions) close() {
	c.devices.Close()
}
