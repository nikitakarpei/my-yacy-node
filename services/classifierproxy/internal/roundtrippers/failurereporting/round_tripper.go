// Package failurereporting reports the step at which the egress round trip
// failed.
package failurereporting

import (
	"net/http"
)

type Egress interface {
	RoundTrip(request *http.Request) (*http.Response, error)
}

type RoundTripper struct {
	egress    Egress
	observers Observers
}

func New(egress Egress, observers Observers) *RoundTripper {
	return &RoundTripper{egress: egress, observers: observers}
}

func (r *RoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	progress := &requestProgress{}
	response, err := r.egress.RoundTrip(
		request.WithContext(progress.tracedContextFrom(request.Context())),
	)
	if err != nil && request.Context().Err() == nil {
		r.observers.RoundTripFailed(
			request.Context(),
			request.URL.String(),
			progress.failedStep(),
			err,
		)
	}
	return response, err
}
