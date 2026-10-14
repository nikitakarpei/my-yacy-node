package webhook_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/messageadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/messageinboxes/webhook"
)

var errUnknownHost = errors.New("unknown host")

type resolverStub map[string][]string

func (r resolverStub) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	raw, known := r[host]
	if !known {
		return nil, errUnknownHost
	}

	addresses := make([]netip.Addr, len(raw))
	for i, address := range raw {
		addresses[i] = netip.MustParseAddr(address)
	}

	return addresses, nil
}

type recordedRequest struct {
	method      string
	host        string
	contentType string
	body        map[string]any
}

type recordingServer struct {
	*httptest.Server

	mu       sync.Mutex
	requests []recordedRequest
}

func newRecordingServer(t *testing.T, status int) *recordingServer {
	t.Helper()

	server := &recordingServer{}
	server.Server = httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorded := recordedRequest{
				method:      r.Method,
				host:        r.Host,
				contentType: r.Header.Get("Content-Type"),
			}
			_ = json.NewDecoder(r.Body).Decode(&recorded.body)

			server.mu.Lock()
			server.requests = append(server.requests, recorded)
			server.mu.Unlock()

			w.WriteHeader(status)
		}),
	)
	t.Cleanup(server.Close)

	return server
}

func (s *recordingServer) received() []recordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]recordedRequest(nil), s.requests...)
}

func (s *recordingServer) port(t *testing.T) string {
	t.Helper()

	_, port, err := net.SplitHostPort(s.Listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener address: %v", err)
	}

	return port
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url %q: %v", raw, err)
	}

	return parsed
}

func sampleMessage() messageadmission.Message {
	return messageadmission.Message{
		Subject: "your peer",
		Body:    "sends me spam",
		Sender: messageadmission.Sender{
			Hash:    yacymodel.WordHash("zeus"),
			Name:    "zeus",
			Address: "203.0.113.7",
		},
	}
}

func TestPrivateWebhookGetsTheMessageDirectly(t *testing.T) {
	t.Parallel()

	inbox := newRecordingServer(t, http.StatusOK)
	proxy := newRecordingServer(t, http.StatusOK)
	target := mustParseURL(t, "http://inbox.test:"+inbox.port(t)+"/hook")

	delivered := webhook.New(
		target,
		mustParseURL(t, proxy.URL),
		resolverStub{"inbox.test": {"127.0.0.1"}},
	).Deliver(context.Background(), sampleMessage())

	if !delivered {
		t.Fatal("message not delivered")
	}
	received := inbox.received()
	if len(received) != 1 {
		t.Fatalf("webhook got %d requests, want 1", len(received))
	}
	if received[0].method != http.MethodPost || received[0].contentType != "application/json" {
		t.Fatalf(
			"request = %s %s, want POST application/json",
			received[0].method,
			received[0].contentType,
		)
	}
	want := map[string]any{
		"subject": "your peer",
		"body":    "sends me spam",
		"sender": map[string]any{
			"hash":    yacymodel.WordHash("zeus").String(),
			"name":    "zeus",
			"address": "203.0.113.7",
		},
	}
	if got, _ := json.Marshal(received[0].body); string(got) != string(mustMarshal(t, want)) {
		t.Fatalf("payload = %s, want %s", got, mustMarshal(t, want))
	}
	if len(proxy.received()) != 0 {
		t.Fatal("private webhook went through the egress proxy")
	}
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	return encoded
}

func TestPrivateWebhookTriesEachResolvedAddress(t *testing.T) {
	t.Parallel()

	inbox := newRecordingServer(t, http.StatusOK)
	target := mustParseURL(t, "http://inbox.test:"+inbox.port(t)+"/hook")

	delivered := webhook.New(
		target,
		mustParseURL(t, "http://127.0.0.1:1"),
		resolverStub{"inbox.test": {"127.0.0.2", "127.0.0.1"}},
	).Deliver(context.Background(), sampleMessage())

	if !delivered || len(inbox.received()) != 1 {
		t.Fatalf(
			"delivered = %v with %d requests, want one delivery",
			delivered,
			len(inbox.received()),
		)
	}
}

func TestWebhookAnswerOutsideSuccessFailsTheDelivery(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusInternalServerError, http.StatusTemporaryRedirect} {
		inbox := newRecordingServer(t, status)
		target := mustParseURL(t, "http://inbox.test:"+inbox.port(t)+"/hook")

		delivered := webhook.New(
			target,
			mustParseURL(t, "http://127.0.0.1:1"),
			resolverStub{"inbox.test": {"127.0.0.1"}},
		).Deliver(context.Background(), sampleMessage())

		if delivered {
			t.Fatalf("status %d: delivered, want failure", status)
		}
	}
}

func TestWebhookRedirectIsNotFollowed(t *testing.T) {
	t.Parallel()

	elsewhere := newRecordingServer(t, http.StatusOK)
	redirecting := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", elsewhere.URL)
			w.WriteHeader(http.StatusTemporaryRedirect)
		}),
	)
	t.Cleanup(redirecting.Close)

	delivered := webhook.New(
		mustParseURL(t, redirecting.URL),
		mustParseURL(t, "http://127.0.0.1:1"),
		resolverStub{"127.0.0.1": {"127.0.0.1"}},
	).Deliver(context.Background(), sampleMessage())

	if delivered || len(elsewhere.received()) != 0 {
		t.Fatalf(
			"delivered = %v, redirect target got %d requests; want neither",
			delivered,
			len(elsewhere.received()),
		)
	}
}

func TestPublicWebhookGoesThroughTheEgressProxy(t *testing.T) {
	t.Parallel()

	proxy := newRecordingServer(t, http.StatusForbidden)

	delivered := webhook.New(
		mustParseURL(t, "https://inbox.example/hook"),
		mustParseURL(t, proxy.URL),
		resolverStub{"inbox.example": {"203.0.113.5"}},
	).Deliver(context.Background(), sampleMessage())

	if delivered {
		t.Fatal("delivered past a refusing proxy")
	}
	received := proxy.received()
	if len(received) != 1 || received[0].method != http.MethodConnect ||
		received[0].host != "inbox.example:443" {
		t.Fatalf("proxy got %#v, want one CONNECT inbox.example:443", received)
	}
}

func TestPublicWebhookWithoutHTTPSIsNotCalled(t *testing.T) {
	t.Parallel()

	cases := map[string][]string{
		"public":             {"203.0.113.5"},
		"private and public": {"127.0.0.1", "203.0.113.5"},
	}
	for name, addresses := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			inbox := newRecordingServer(t, http.StatusOK)
			proxy := newRecordingServer(t, http.StatusOK)
			target := mustParseURL(t, "http://inbox.test:"+inbox.port(t)+"/hook")

			delivered := webhook.New(
				target,
				mustParseURL(t, proxy.URL),
				resolverStub{"inbox.test": addresses},
			).Deliver(context.Background(), sampleMessage())

			if delivered || len(inbox.received()) != 0 || len(proxy.received()) != 0 {
				t.Fatalf("delivered = %v, webhook %d, proxy %d requests; want no call",
					delivered, len(inbox.received()), len(proxy.received()))
			}
		})
	}
}

func TestUnresolvedWebhookFailsTheDelivery(t *testing.T) {
	t.Parallel()

	delivered := webhook.New(
		mustParseURL(t, "https://inbox.example/hook"),
		mustParseURL(t, "http://127.0.0.1:1"),
		resolverStub{},
	).Deliver(context.Background(), sampleMessage())

	if delivered {
		t.Fatal("delivered to an unresolved host")
	}
}

func TestPrivateWebhookWithoutAnAnsweringAddressFailsTheDelivery(t *testing.T) {
	t.Parallel()

	delivered := webhook.New(
		mustParseURL(t, "http://inbox.test:1/hook"),
		mustParseURL(t, "http://127.0.0.1:1"),
		resolverStub{"inbox.test": {"127.0.0.2", "127.0.0.3"}},
	).Deliver(context.Background(), sampleMessage())

	if delivered {
		t.Fatal("delivered with no address answering")
	}
}
