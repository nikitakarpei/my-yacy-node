package main

import (
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodestatus"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiingress"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

func wireRouterOn(
	mux *http.ServeMux,
	config nodeconfiguration.ServingConfig,
	status nodestatus.RuntimeStatus,
) httpguard.WireRouter {
	return httpguard.NewWireRouter(mux, httpguard.WireGate{
		Guard: httpguard.NewRequestGuard(
			httpguard.DefaultMaxBodyBytes,
			httpguard.DefaultRequestTimeout,
		),
		Respond: httpguard.NewWireResponder(status),
		Address: httpguard.NewClientAddressResolver(config.TrustedProxyNetworks),
	})
}

const searchPostingsPerWord = 1000

func mountNodeEndpoints(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	storage nodeStorage,
	searchObserver *searchmetrics.SearchMetrics,
	partitions yacymodel.DHTRingPartitions,
) {
	urlmeta.MountTransferURL(router, identity, storage.urlReceiver)
	rwiingress.Mount(router, identity, storage.postingReceiver)
	nodestatus.MountQuery(
		router,
		identity,
		storage.postings,
		storage.references,
		storage.urlDirectory,
	)

	documentsearch.MountSearch(
		router,
		identity,
		storage.postings,
		storage.urlDirectory,
		searchPostingsPerWord,
		searchObserver,
		partitions,
	)
}
