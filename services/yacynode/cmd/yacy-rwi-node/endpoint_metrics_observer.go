package main

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
)

type endpointMetricsObserver struct {
	endpoints *metrics.HTTPEndpointMetrics
}

func (o endpointMetricsObserver) ObserveRequest(
	_ context.Context,
	served httpobservation.ServedRequest,
) {
	o.endpoints.Observe(served.Pattern, served.Status, served.Duration)
}
