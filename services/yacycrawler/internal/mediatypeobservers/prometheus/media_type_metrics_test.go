package prometheus_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	mediatypeobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/mediatypeobservers/prometheus"
)

const metricName = "yacycrawler_page_media_types_unparsed_total"

func TestMediaTypeMetricsCountEveryContentTypeThatCannotBeParsed(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := mediatypeobserversprometheus.New(registry)

	metrics.MediaTypeUnparsed(context.Background(), "text/html; charset", errors.New("mime"))
	metrics.MediaTypeUnparsed(context.Background(), "text/html;;", errors.New("mime"))

	if err := testutil.GatherAndCompare(registry, strings.NewReader(
		"# HELP "+metricName+" Fetched pages whose content type cannot be parsed.\n"+
			"# TYPE "+metricName+" counter\n"+
			metricName+" 2\n",
	), metricName); err != nil {
		t.Fatal(err)
	}
}

func TestAnUnparsedContentTypeReadsZeroBeforeItHappens(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	mediatypeobserversprometheus.New(registry)

	if counters := testutil.CollectAndCount(registry, metricName); counters != 1 {
		t.Fatalf("counters = %d, want 1", counters)
	}
}
