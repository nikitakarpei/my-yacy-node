package publicaddress

import (
	"context"
	"log/slog"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type observedAddress struct {
	mu   sync.Mutex
	host yacymodel.Optional[yacymodel.Host]
}

func (a *observedAddress) Host(context.Context) yacymodel.Optional[yacymodel.Host] {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.host
}

func (a *observedAddress) Settle(ctx context.Context, observedHosts []yacymodel.Host) bool {
	agreement := agreementAmong(observedHosts)
	if !agreement.reached() {
		a.reportDisagreement(ctx, agreement)

		return false
	}

	return a.adopt(ctx, agreement)
}

func (a *observedAddress) reportDisagreement(ctx context.Context, agreement hostAgreement) {
	level := slog.LevelDebug
	if !a.Host(ctx).Present() {
		level = slog.LevelWarn
	}
	slog.Log(ctx, level, "peers did not agree on the public address",
		slog.Int("agreeingPeers", agreement.agreeingPeers),
		slog.Int("reportingPeers", agreement.reportingPeers),
	)
}

func (a *observedAddress) adopt(ctx context.Context, agreement hostAgreement) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if current, present := a.host.Get(); present && current == agreement.host {
		return false
	}
	a.host = yacymodel.Some(agreement.host)
	slog.InfoContext(ctx, "public address learned from peers",
		slog.String("host", agreement.host.String()),
		slog.Int("agreeingPeers", agreement.agreeingPeers),
		slog.Int("reportingPeers", agreement.reportingPeers),
	)

	return true
}
