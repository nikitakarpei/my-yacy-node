//go:build e2e

package nodepeer

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerurl"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
)

const opsExposedPort = "9090/tcp"

func SeriesValueOf(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	node testcontainers.Container,
	series string,
) (float64, bool) {
	t.Helper()
	result := probe.Get(ctx, containerurl.HostURL(t, ctx, node, opsExposedPort)+"/metrics")
	if !result.OK {
		return 0, false
	}
	for line := range strings.SplitSeq(result.Body, "\n") {
		if valueText, found := strings.CutPrefix(line, series+" "); found {
			value, err := strconv.ParseFloat(valueText, 64)
			return value, err == nil
		}
	}
	return 0, false
}

func SeriesSumOf(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	node testcontainers.Container,
	metric string,
) float64 {
	t.Helper()
	var sum float64
	for _, line := range MetricLinesOf(t, ctx, probe, node, metric) {
		series, valueText, found := strings.Cut(line, " ")
		if !found || (series != metric && !strings.HasPrefix(series, metric+"{")) {
			continue
		}
		if value, err := strconv.ParseFloat(valueText, 64); err == nil {
			sum += value
		}
	}
	return sum
}

func MetricLinesOf(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	node testcontainers.Container,
	prefix string,
) []string {
	t.Helper()
	result := probe.Get(ctx, containerurl.HostURL(t, ctx, node, opsExposedPort)+"/metrics")
	if !result.OK {
		return nil
	}
	var lines []string
	for line := range strings.SplitSeq(result.Body, "\n") {
		if strings.HasPrefix(line, prefix) {
			lines = append(lines, line)
		}
	}
	return lines
}
