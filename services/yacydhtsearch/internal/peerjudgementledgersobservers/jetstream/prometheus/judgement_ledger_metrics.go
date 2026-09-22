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
	actionLookup = "lookup"
	actionHold   = "hold"
)

type JudgementLedgerMetrics struct {
	lookupFailures prometheusclient.Counter
	holdFailures   prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *JudgementLedgerMetrics {
	failures := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_peer_judgement_ledger_failures_total",
		Help: "Failures against the ledger of the peer judgements, by action.",
	}, []string{labelAction})
	registry.MustRegister(failures)

	return &JudgementLedgerMetrics{
		lookupFailures: failures.WithLabelValues(actionLookup),
		holdFailures:   failures.WithLabelValues(actionHold),
	}
}

func (m *JudgementLedgerMetrics) JudgementLookupFailed(
	context.Context,
	yacymodel.Hash,
	peerjudgements.Question,
	error,
) {
	m.lookupFailures.Inc()
}

func (m *JudgementLedgerMetrics) JudgementHoldFailed(
	context.Context,
	yacymodel.Hash,
	peerjudgements.Question,
	error,
) {
	m.holdFailures.Inc()
}
