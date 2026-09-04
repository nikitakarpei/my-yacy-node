package httpobservation_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
)

type recordedRequests struct {
	served []httpobservation.ServedRequest
}

func (r *recordedRequests) ObserveRequest(
	_ context.Context,
	served httpobservation.ServedRequest,
) {
	r.served = append(r.served, served)
}

func serve(t *testing.T, next http.Handler, observers ...httpobservation.Observer) {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle("GET /posting/{hash}", next)

	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/posting/abc", nil,
	)
	httpobservation.NewHandler(mux, observers...).ServeHTTP(response, request)
}

func TestEveryObserverSeesTheServedRequest(t *testing.T) {
	first := &recordedRequests{}
	second := &recordedRequests{}

	serve(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}), first, second)

	for name, observer := range map[string]*recordedRequests{"first": first, "second": second} {
		if len(observer.served) != 1 {
			t.Fatalf("%s observer saw %d requests, want 1", name, len(observer.served))
		}
		served := observer.served[0]
		if served.Status != http.StatusTeapot {
			t.Errorf("%s status = %d, want %d", name, served.Status, http.StatusTeapot)
		}
		if served.Method != http.MethodGet || served.Path != "/posting/abc" {
			t.Errorf("%s request = %s %s", name, served.Method, served.Path)
		}
		if served.Pattern != "GET /posting/{hash}" {
			t.Errorf("%s pattern = %q, want the route pattern", name, served.Pattern)
		}
	}
}

func TestAHandlerThatWritesNoStatusIsServedAsOK(t *testing.T) {
	observer := &recordedRequests{}

	serve(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte("body")); err != nil {
			t.Errorf("write body: %v", err)
		}
	}), observer)

	if observer.served[0].Status != http.StatusOK {
		t.Errorf("status = %d, want %d", observer.served[0].Status, http.StatusOK)
	}
}

func TestAnInformationalStatusIsNotTheServedStatus(t *testing.T) {
	observer := &recordedRequests{}

	serve(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusEarlyHints)
		w.WriteHeader(http.StatusCreated)
	}), observer)

	if observer.served[0].Status != http.StatusCreated {
		t.Errorf("status = %d, want the status that ended the response", observer.served[0].Status)
	}
}

type failingResponseWriter struct {
	header http.Header
}

func (w *failingResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}

func (*failingResponseWriter) WriteHeader(int) {}

func (*failingResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("client gone")
}

func TestAResponseWriteFailureIsObserved(t *testing.T) {
	observer := &recordedRequests{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("body"))
	})
	request := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/", nil,
	)

	httpobservation.NewHandler(handler, observer).ServeHTTP(&failingResponseWriter{}, request)

	if observer.served[0].ResponseWriteError == nil {
		t.Fatal("response write error was not observed")
	}
}
