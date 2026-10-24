package peerannouncement

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ReportedHostObserver interface {
	ObserveReportedHosts(ctx context.Context, reportedHosts []yacymodel.Host)
}

type discardReportedHostObserver struct{}

func (discardReportedHostObserver) ObserveReportedHosts(context.Context, []yacymodel.Host) {}

var DiscardReportedHostObserver ReportedHostObserver = discardReportedHostObserver{}
