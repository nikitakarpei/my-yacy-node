package publicaddress

import (
	"context"
	"log/slog"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type peerSource struct {
	mu   sync.Mutex
	host yacymodel.Optional[yacymodel.Host]
}

func (s *peerSource) Host(context.Context) yacymodel.Optional[yacymodel.Host] {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.host
}

func (s *peerSource) ObserveReportedHosts(ctx context.Context, reportedHosts []yacymodel.Host) {
	agreement := agreementAmong(reportedHosts)
	if !agreement.reached() {
		s.reportDisagreement(ctx, agreement)

		return
	}

	s.adopt(ctx, agreement)
}

func (s *peerSource) reportDisagreement(ctx context.Context, agreement hostAgreement) {
	level := slog.LevelDebug
	if !s.Host(ctx).Present() {
		level = slog.LevelWarn
	}
	slog.Log(ctx, level, "peers did not agree on the public address",
		slog.Int("agreeingPeers", agreement.agreeingPeers),
		slog.Int("reportingPeers", agreement.reportingPeers),
	)
}

func (s *peerSource) adopt(ctx context.Context, agreement hostAgreement) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if current, present := s.host.Get(); present && current == agreement.host {
		return
	}
	s.host = yacymodel.Some(agreement.host)
	slog.InfoContext(ctx, "public address learned from peers",
		slog.String("host", agreement.host.String()),
		slog.Int("agreeingPeers", agreement.agreeingPeers),
		slog.Int("reportingPeers", agreement.reportingPeers),
	)
}
