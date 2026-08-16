package main

import (
	"net"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/landing"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodestatus"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiingress"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

func wireRouterOn(
	mux *http.ServeMux,
	trustedProxyNetworks []*net.IPNet,
	status nodestatus.RuntimeStatus,
) httpguard.WireRouter {
	return httpguard.NewWireRouter(mux, httpguard.WireGate{
		Guard: httpguard.NewRequestGuard(
			httpguard.DefaultMaxBodyBytes,
			httpguard.DefaultRequestTimeout,
		),
		Respond: httpguard.NewWireResponder(status),
		Address: httpguard.NewClientAddressResolver(trustedProxyNetworks),
	})
}

const searchPostingsPerWord = 1000

type nodeEndpoints struct {
	mux            *http.ServeMux
	router         httpguard.WireRouter
	identity       nodeidentity.Identity
	storage        nodeStorage
	searchObserver *searchmetrics.SearchMetrics
	partitions     yacymodel.DHTRingPartitions
}

func (e nodeEndpoints) mount() {
	e.mux.Handle("/{$}", landing.NewEndpoint())

	urlmeta.MountTransferURL(e.router, e.identity, e.storage.urlReceiver)
	rwiingress.Mount(e.router, e.identity, e.storage.postingReceiver)
	nodestatus.MountQuery(
		e.router,
		e.identity,
		e.storage.postings,
		e.storage.references,
		e.storage.urlDirectory,
	)

	documentsearch.MountSearch(
		e.router,
		e.identity,
		e.storage.postings,
		e.storage.urlDirectory,
		searchPostingsPerWord,
		e.searchObserver,
		e.partitions,
	)
}
