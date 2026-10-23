package prometheus_test

import (
	"context"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	crawlobserversfrontierprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/frontier/prometheus"
)

type frontier struct {
	amount  int64
	counted bool
}

func (f frontier) AmountPending(context.Context) (int64, bool) {
	return f.amount, f.counted
}

func TestPendingPageVisitsReportTheAmountOnTheFrontier(t *testing.T) {
	registry := prometheus.NewRegistry()
	crawlobserversfrontierprometheus.New(registry, frontier{amount: 42, counted: true})

	expected := `
# HELP yacycrawler_pending_page_visits Page visits pending on the frontier, leased or not.
# TYPE yacycrawler_pending_page_visits gauge
yacycrawler_pending_page_visits 42
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected)); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestPendingPageVisitsAreLeftOutWhenTheFrontierCannotCountThem(t *testing.T) {
	registry := prometheus.NewRegistry()
	crawlobserversfrontierprometheus.New(registry, frontier{})

	if amount, err := testutil.GatherAndCount(registry); err != nil || amount != 0 {
		t.Fatalf("GatherAndCount = %d, %v; want no sample", amount, err)
	}
}
