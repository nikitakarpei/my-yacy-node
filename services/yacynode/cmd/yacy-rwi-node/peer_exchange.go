package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/bootstrap"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodestatus"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peeradmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerannouncement"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type peerExchange struct {
	router         httpguard.WireRouter
	identity       nodeidentity.Identity
	status         nodestatus.RuntimeStatus
	config         nodeconfiguration.PeerExchangeConfig
	vault          *vault.Vault
	now            func() time.Time
	client         *http.Client
	rosterObserver peerroster.RosterObserver
}

func (p peerExchange) assemble() (peerannouncement.Announcer, peerroster.Roster, error) {
	roster, err := peerroster.Open(
		p.vault,
		p.now,
		p.config.KnownRosterCapacity,
		p.config.ReachableRosterCapacity,
		p.config.AnnounceInterval,

		p.identity.Hash,
		p.rosterObserver,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("open peer roster: %w", err)
	}

	peeradmission.MountHello(
		p.router,
		p.identity,
		peerAdmissionStatus{status: p.status, networkName: p.identity.NetworkName},
		roster,
		p.client,
	)

	announcer := peerannouncement.New(
		peerannouncement.Config{
			Client:             p.client,
			NetworkName:        p.identity.NetworkName,
			Interval:           p.config.AnnounceInterval,
			ReachableCap:       p.config.ReachableRosterCapacity,
			ContactConcurrency: p.config.PeerContactConcurrency,
		},
		p.status,
		bootstrap.New(p.client, p.config.SeedlistURLs),
		roster,
	)

	return announcer, roster, nil
}
