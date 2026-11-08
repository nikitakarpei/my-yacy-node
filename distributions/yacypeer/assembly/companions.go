package yacypeerassembly

import (
	"crypto/tls"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/brokerkey"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionaccess"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/devicepermissions"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/paireddevices"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/pairingoffers"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
)

const (
	companionsDirectoryName = "companions"
	pairingSecretLifetime   = time.Minute
	pairingWait             = 30 * time.Second
)

type companions interface {
	startBroker(directory string) (*natsbroker.Broker, error)
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

func (c *servedCompanions) close() {
	c.devices.Close()
}
