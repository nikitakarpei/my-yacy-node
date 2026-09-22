// Package prometheus reports how often the ledger of the peer judgements
// failed.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	labelAction  = "action"
	actionQueue  = "queue"
	actionHold   = "hold"
	actionWatch  = "watch"
	actionDecode = "decode"
)

type JudgementLedgerMetrics struct {
	queueFailures  prometheusclient.Counter
	holdFailures   prometheusclient.Counter
	watchFailures  prometheusclient.Counter
	decodeFailures prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *JudgementLedgerMetrics {
	failures := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_peer_judgement_ledger_failures_total",
		Help: "Failures against the ledger of the peer judgements, by action.",
	}, []string{labelAction})
	registry.MustRegister(failures)

	return &JudgementLedgerMetrics{
		queueFailures:  failures.WithLabelValues(actionQueue),
		holdFailures:   failures.WithLabelValues(actionHold),
		watchFailures:  failures.WithLabelValues(actionWatch),
		decodeFailures: failures.WithLabelValues(actionDecode),
	}
}

func (m *JudgementLedgerMetrics) JudgementDropped(
	context.Context,
	yacymodel.Hash,
	peerjudgements.Question,
) {
	m.queueFailures.Inc()
}

func (m *JudgementLedgerMetrics) JudgementHoldFailed(
	context.Context,
	yacymodel.Hash,
	peerjudgements.Question,
	error,
) {
	m.holdFailures.Inc()
}

func (m *JudgementLedgerMetrics) WatchFailed(context.Context, error) {
	m.watchFailures.Inc()
}

func (m *JudgementLedgerMetrics) JudgementUndecodable(context.Context, string, error) {
	m.decodeFailures.Inc()
}

func (m *JudgementLedgerMetrics) WatchEnded(context.Context) {
	m.watchFailures.Inc()
}
