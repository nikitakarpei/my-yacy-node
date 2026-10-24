package yacynodeassembly

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodestatus"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerannouncement"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/publicaddress"
)

func publicAddressFor(
	statedHost yacymodel.Optional[yacymodel.Host],
) (nodestatus.PublicAddressSource, peerannouncement.ReportedHostObserver) {
	if host, stated := statedHost.Get(); stated {
		return publicaddress.Stated(host), peerannouncement.DiscardReportedHostObserver
	}
	peerSource := &publicaddress.PeerSource{}

	return peerSource, peerSource
}
