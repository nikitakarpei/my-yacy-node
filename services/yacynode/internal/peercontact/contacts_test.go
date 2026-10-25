package peercontact_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peercontact"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	advertisedPort          = yacymodel.Port(8090)
	securePort              = yacymodel.Port(8443)
	answeredOriginsCapacity = 16
)

var errNothingListens = errors.New("nothing listens")

type stoppedClock struct{}

func (stoppedClock) Now() time.Time { return time.Unix(1_700_000_000, 0).UTC() }

func (stoppedClock) After(time.Duration, func()) (stop func()) { return func() {} }

type network struct {
	guard  sync.Mutex
	routes map[string]string
	dialed []string
	tls    *httptest.Server
}

func newNetwork(t *testing.T) *network {
	t.Helper()

	return &network{routes: make(map[string]string)}
}

func (n *network) answer(t *testing.T, address string, reply string) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(server.Close)
	n.route(address, server.Listener.Addr().String())
}

func (n *network) answerSecurely(t *testing.T, address string, reply string) {
	t.Helper()

	n.tls = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(n.tls.Close)
	n.route(address, n.tls.Listener.Addr().String())
}

func (n *network) route(address, target string) {
	n.guard.Lock()
	defer n.guard.Unlock()

	n.routes[address] = target
}

func (n *network) dial(ctx context.Context, protocol, address string) (net.Conn, error) {
	n.guard.Lock()
	n.dialed = append(n.dialed, address)
	target, routed := n.routes[address]
	n.guard.Unlock()

	if !routed {
		return nil, errNothingListens
	}

	var dialer net.Dialer

	conn, err := dialer.DialContext(ctx, protocol, target)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	return conn, nil
}

func (n *network) dialedAddresses() []string {
	n.guard.Lock()
	defer n.guard.Unlock()

	dialed := slices.Clone(n.dialed)
	n.dialed = nil

	return dialed
}

func (n *network) contacts() *peercontact.Contacts {
	return n.contactsKeeping(answeredOriginsCapacity)
}

func (n *network) contactsKeeping(capacity int) *peercontact.Contacts {
	roots := x509.NewCertPool()
	if n.tls != nil {
		roots.AddCert(n.tls.Certificate())
	}
	client := &http.Client{Transport: &http.Transport{
		DialContext:       n.dial,
		TLSClientConfig:   &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
		DisableKeepAlives: true,
	}}

	return peercontact.New(peerwire.NewFormExchange(client, stoppedClock{}), capacity)
}

func hostOf(t *testing.T, s string) yacymodel.Host {
	t.Helper()

	host, err := yacymodel.ParseHost(s)
	if err != nil {
		t.Fatalf("parse host: %v", err)
	}

	return host
}

func addressesOf(t *testing.T, hosts ...string) yacymodel.PeerAddresses {
	t.Helper()

	addresses := yacymodel.PeerAddresses{Port: advertisedPort}
	for _, host := range hosts {
		addresses.Hosts = append(addresses.Hosts, hostOf(t, host))
	}

	return addresses
}

func peerHash(t *testing.T, s string) yacymodel.Hash {
	t.Helper()

	hash, err := yacymodel.ParseHash(s)
	if err != nil {
		t.Fatalf("parse hash: %v", err)
	}

	return hash
}

func hasField(field string) func(yacyproto.ResponseFields) bool {
	return func(fields yacyproto.ResponseFields) bool { return fields[field] != "" }
}

func questionFor(peer yacymodel.Hash) peercontact.Question {
	return peercontact.Question{
		Request: peerwire.Request{Peer: peer, Path: "/yacy/hello.html"},
		Accepts: hasField("identity"),
	}
}

func TestGreetTriesHTTPSOnADomainNameFirst(t *testing.T) {
	web := newNetwork(t)
	web.answerSecurely(t, "example.com:443", "identity=peer\n")
	web.answer(t, "example.com:8090", "identity=peer\n")

	answer, answered := web.contacts().Greet(
		t.Context(),
		addressesOf(t, "example.com"),
		questionFor(peerHash(t, "AAAAAAAAAAAA")),
	)

	if !answered || answer.Origin.Scheme != peerwire.HTTPS || answer.Origin.Port != 443 {
		t.Fatalf("answer = %v, %v, want https on port 443", answer.Origin, answered)
	}
}

func TestGreetWalksEveryOriginOfEveryHostInOrder(t *testing.T) {
	web := newNetwork(t)
	addresses := addressesOf(t, "example.com", "192.0.2.1", "198.51.100.1")
	addresses.SecurePort = yacymodel.Some(securePort)

	_, answered := web.contacts().Greet(
		t.Context(),
		addresses,
		questionFor(peerHash(t, "AAAAAAAAAAAA")),
	)

	want := []string{
		"example.com:443", "example.com:8443", "example.com:8090",
		"192.0.2.1:8090", "198.51.100.1:8090",
	}
	if dialed := web.dialedAddresses(); answered || !slices.Equal(dialed, want) {
		t.Fatalf("dialed = %v, want %v", dialed, want)
	}
}

func TestGreetAsksAtMostFiveOrigins(t *testing.T) {
	web := newNetwork(t)

	web.contacts().Greet(
		t.Context(),
		addressesOf(t, "a.example", "b.example", "c.example", "d.example", "e.example"),
		questionFor(peerHash(t, "AAAAAAAAAAAA")),
	)

	if dialed := web.dialedAddresses(); len(dialed) != 5 {
		t.Fatalf("dialed = %v, want five origins", dialed)
	}
}

func TestGreetNeedsAnAcceptedReply(t *testing.T) {
	web := newNetwork(t)
	web.answer(t, "192.0.2.1:8090", "reply=yes\n")

	_, answered := web.contacts().Greet(
		t.Context(),
		addressesOf(t, "192.0.2.1"),
		questionFor(peerHash(t, "AAAAAAAAAAAA")),
	)

	if answered {
		t.Fatal("a reply that does not identify the peer answered the greet")
	}
}

func TestProbeLeadsWithTheOriginThatAnsweredTheLastGreet(t *testing.T) {
	web := newNetwork(t)
	web.answer(t, "198.51.100.1:8090", "identity=peer\nreply=yes\n")
	contacts := web.contacts()
	peer := peerHash(t, "AAAAAAAAAAAA")
	addresses := addressesOf(t, "192.0.2.1", "198.51.100.1")
	contacts.Greet(t.Context(), addresses, questionFor(peer))
	web.dialedAddresses()

	_, answered := contacts.Probe(t.Context(), addresses, questionFor(peer))

	if dialed := web.dialedAddresses(); !answered ||
		!slices.Equal(dialed, []string{"198.51.100.1:8090"}) {
		t.Fatalf("dialed = %v, %v, want the answering origin only", dialed, answered)
	}
}

func TestGreetRemembersTheOriginThatAnswered(t *testing.T) {
	web := newNetwork(t)
	web.answer(t, "198.51.100.1:8090", "identity=peer\n")
	contacts := web.contacts()
	peer := peerHash(t, "AAAAAAAAAAAA")

	contacts.Greet(t.Context(), addressesOf(t, "192.0.2.1", "198.51.100.1"), questionFor(peer))

	if origin, found := contacts.AnsweredOriginOf(peer); !found ||
		origin.Host != hostOf(t, "198.51.100.1") {
		t.Fatalf("answered origin = %v, %v, want 198.51.100.1", origin, found)
	}
}

func TestProbeRemembersTheOriginThatAnswered(t *testing.T) {
	web := newNetwork(t)
	web.answer(t, "198.51.100.1:8090", "identity=peer\n")
	contacts := web.contacts()
	peer := peerHash(t, "AAAAAAAAAAAA")

	contacts.Probe(t.Context(), addressesOf(t, "192.0.2.1", "198.51.100.1"), questionFor(peer))

	if origin, found := contacts.AnsweredOriginOf(peer); !found ||
		origin.Host != hostOf(t, "198.51.100.1") {
		t.Fatalf("answered origin = %v, %v, want 198.51.100.1", origin, found)
	}
}

func TestGreetWithoutAnswerForgetsTheOrigin(t *testing.T) {
	web := newNetwork(t)
	web.answer(t, "198.51.100.1:8090", "identity=peer\n")
	contacts := web.contacts()
	peer := peerHash(t, "AAAAAAAAAAAA")
	contacts.Greet(t.Context(), addressesOf(t, "198.51.100.1"), questionFor(peer))

	contacts.Greet(t.Context(), addressesOf(t, "203.0.113.1"), questionFor(peer))

	if origin, found := contacts.AnsweredOriginOf(peer); found {
		t.Fatalf("answered origin = %v, want none after a greet without answer", origin)
	}
}

func TestDeliverAsksOnlyTheGivenOrigin(t *testing.T) {
	web := newNetwork(t)
	web.answer(t, "198.51.100.1:8090", "reply=yes\n")
	origin := peerwire.Origin{
		Scheme: peerwire.HTTP,
		Host:   hostOf(t, "198.51.100.1"),
		Port:   advertisedPort,
	}

	answer, answered := web.contacts().Deliver(t.Context(), origin, peercontact.Question{
		Request: peerwire.Request{
			Peer: peerHash(t, "AAAAAAAAAAAA"),
			Path: "/yacy/transferRWI.html",
		},
		Accepts: hasField("reply"),
	})

	if dialed := web.dialedAddresses(); !answered || answer.Origin != origin ||
		!slices.Equal(dialed, []string{"198.51.100.1:8090"}) {
		t.Fatalf("dialed = %v, %v, want only the given origin", dialed, answered)
	}
}

func TestContactsKeepAtMostCapacityAnsweredOrigins(t *testing.T) {
	web := newNetwork(t)
	web.answer(t, "192.0.2.1:8090", "identity=peer\n")
	web.answer(t, "198.51.100.1:8090", "identity=peer\n")
	contacts := web.contactsKeeping(1)
	first := peerHash(t, "AAAAAAAAAAAA")
	second := peerHash(t, "BBBBBBBBBBBB")

	contacts.Greet(t.Context(), addressesOf(t, "192.0.2.1"), questionFor(first))
	contacts.Greet(t.Context(), addressesOf(t, "198.51.100.1"), questionFor(second))

	if _, found := contacts.AnsweredOriginOf(first); found {
		t.Fatal("the first peer's origin kept beyond a capacity of one")
	}
	if _, found := contacts.AnsweredOriginOf(second); !found {
		t.Fatal("the second peer's origin not kept")
	}
}
