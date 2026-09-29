// Package upstreamrequest sends requests to the egress proxy and reports the
// step at which a request fails.
package upstreamrequest

import (
	"net/http"
)

type Egress interface {
	RoundTrip(request *http.Request) (*http.Response, error)
}

type Sender struct {
	egress    Egress
	observers Observers
}

func New(egress Egress, observers Observers) *Sender {
	return &Sender{egress: egress, observers: observers}
}

func (s *Sender) RoundTrip(request *http.Request) (*http.Response, error) {
	progress := &requestProgress{}
	response, err := s.egress.RoundTrip(
		request.WithContext(progress.tracedContextFrom(request.Context())),
	)
	if err != nil && request.Context().Err() == nil {
		s.observers.Failed(request.Context(), request.URL.String(), progress.failedStep(), err)
	}
	return response, err
}
