package peerport_test

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerport"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type schemeRecorder struct {
	mu      sync.Mutex
	schemes []peerport.Scheme
}

func (r *schemeRecorder) ObserveConnection(scheme peerport.Scheme) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.schemes = append(r.schemes, scheme)
}

func (r *schemeRecorder) observedSchemes() []peerport.Scheme {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.schemes)
}

var (
	farFutureClock = fixedClock{now: time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)}
	longPastClock  = fixedClock{now: time.Unix(0, 0)}
)

func TestListenersOnServeHTTPAndHTTPSOnOnePort(t *testing.T) {
	peerPort := peerPortFor(t)
	recorder := &schemeRecorder{}
	listeners := peerport.ListenersOn(t.Context(), peerPort, farFutureClock, recorder)
	httpServer := serverOn(listeners.HTTP)
	httpServer.Start()
	t.Cleanup(httpServer.Close)
	httpsServer := serverOn(listeners.HTTPS)
	httpsServer.StartTLS()
	t.Cleanup(httpsServer.Close)

	address := peerPort.Addr().String()
	if body := bodyOf(t, http.DefaultClient, "http://"+address); body != "http" {
		t.Errorf("http body = %q, want http", body)
	}
	if body := bodyOf(t, httpsServer.Client(), "https://"+address); body != "https" {
		t.Errorf("https body = %q, want https", body)
	}
	want := []peerport.Scheme{peerport.HTTPScheme, peerport.HTTPSScheme}
	if observed := recorder.observedSchemes(); !slices.Equal(observed, want) {
		t.Errorf("observed schemes = %v, want %v", observed, want)
	}
}

func TestListenersOnCloseAConnectionThatSendsNoByteInTime(t *testing.T) {
	peerPort := peerPortFor(t)
	listeners := peerport.ListenersOn(t.Context(), peerPort, longPastClock, &schemeRecorder{})
	t.Cleanup(func() {
		_ = listeners.HTTP.Close()
		_ = listeners.HTTPS.Close()
	})

	connection, err := (&net.Dialer{}).DialContext(t.Context(), "tcp", peerPort.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })

	if _, err := connection.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Errorf("read = %v, want EOF from a closed connection", err)
	}
}

func TestListenersOnKeepServingHTTPSWhileTheHTTPListenerIsClosed(t *testing.T) {
	peerPort := peerPortFor(t)
	listeners := peerport.ListenersOn(t.Context(), peerPort, farFutureClock, &schemeRecorder{})
	httpsServer := serverOn(listeners.HTTPS)
	httpsServer.StartTLS()
	t.Cleanup(httpsServer.Close)

	if err := listeners.HTTP.Close(); err != nil {
		t.Fatalf("close the http listener: %v", err)
	}

	if _, err := listeners.HTTP.Accept(); !errors.Is(err, net.ErrClosed) {
		t.Errorf("http accept = %v, want %v", err, net.ErrClosed)
	}
	if body := bodyOf(
		t,
		httpsServer.Client(),
		"https://"+peerPort.Addr().String(),
	); body != "https" {
		t.Errorf("https body = %q, want https", body)
	}
}

func TestListenersOnReleaseThePeerPortWhenBothListenersClose(t *testing.T) {
	peerPort := peerPortFor(t)
	listeners := peerport.ListenersOn(t.Context(), peerPort, farFutureClock, &schemeRecorder{})

	for _, listener := range []net.Listener{listeners.HTTP, listeners.HTTPS, listeners.HTTPS} {
		if err := listener.Close(); err != nil {
			t.Fatalf("close: %v", err)
		}
	}

	reopened, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", peerPort.Addr().String())
	if err != nil {
		t.Fatalf("listen on the released peer port: %v", err)
	}
	_ = reopened.Close()
}

func TestListenersOnShareTheAddressOfThePeerPort(t *testing.T) {
	peerPort := peerPortFor(t)
	listeners := peerport.ListenersOn(t.Context(), peerPort, farFutureClock, &schemeRecorder{})
	t.Cleanup(func() {
		_ = listeners.HTTP.Close()
		_ = listeners.HTTPS.Close()
	})

	for _, listener := range []net.Listener{listeners.HTTP, listeners.HTTPS} {
		if listener.Addr() != peerPort.Addr() {
			t.Errorf("addr = %s, want %s", listener.Addr(), peerPort.Addr())
		}
	}
}

func peerPortFor(t *testing.T) net.Listener {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	return listener
}

func serverOn(listener net.Listener) *httptest.Server {
	server := httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.TLS != nil {
				_, _ = io.WriteString(w, "https")

				return
			}
			_, _ = io.WriteString(w, "http")
		},
	))
	_ = server.Listener.Close()
	server.Listener = listener

	return server
}

func bodyOf(t *testing.T, client *http.Client, url string) string {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("build request for %s: %v", url, err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("get %s: %v", url, err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s: %v", url, err)
	}

	return string(body)
}
