// Package httpobservation reports how each request was served. It times the
// handler it wraps, keeps the status code the handler wrote, and hands both to
// every observer, so a request is timed once however many observers watch it.
package httpobservation

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type ServedRequest struct {
	Method             string
	Path               string
	Pattern            string
	Status             int
	Duration           time.Duration
	ResponseWriteError error
}

type Observer interface {
	ObserveRequest(ctx context.Context, served ServedRequest)
}

func NewHandler(next http.Handler, observers ...Observer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)

		served := ServedRequest{
			Method:             r.Method,
			Path:               r.URL.Path,
			Pattern:            r.Pattern,
			Status:             recorder.servedStatus(),
			Duration:           time.Since(started),
			ResponseWriteError: recorder.responseWriteError,
		}
		for _, observer := range observers {
			observer.ObserveRequest(r.Context(), served)
		}
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status             int
	responseWriteError error
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status == 0 && status >= http.StatusOK {
		r.status = status
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	written, err := r.ResponseWriter.Write(body)
	if err == nil {
		return written, nil
	}
	responseWriteError := fmt.Errorf("write HTTP response: %w", err)
	if r.responseWriteError == nil {
		r.responseWriteError = responseWriteError
	}
	return written, responseWriteError
}

func (r *statusRecorder) servedStatus() int {
	if r.status == 0 {
		return http.StatusOK
	}

	return r.status
}
