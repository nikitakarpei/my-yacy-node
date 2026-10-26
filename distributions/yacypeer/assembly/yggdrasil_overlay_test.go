package yacypeerassembly_test

import (
	"context"
	"crypto/ed25519"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/yggdrasil-network/yggdrasil-go/src/address"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	yacypeerassembly "github.com/nikitakarpei/yacy-rwi-node/yacypeer/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/ipv6stack"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilkey"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilnode"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	neighbourSeedlistPort = 8080
	overlayDeadline       = 30 * time.Second
	helloFormCeiling      = 1 << 20
)

type yggdrasilNeighbour struct {
	stack   *ipv6stack.Stack
	address netip.Addr
}

func peerOnYggdrasil(t *testing.T) (map[string]string, *url.URL, netip.Addr) {
	t.Helper()
	env := peerEnvironment(t)
	peeringURL := &url.URL{Scheme: "unix", Path: filepath.Join(t.TempDir(), "ygg.sock")}
	env[yacypeerassembly.EnvYggdrasilListen] = peeringURL.String()
	key, err := yggdrasilkey.Settle(
		filepath.Join(env[yacypeerassembly.EnvDataDirectory], "yggdrasil"),
	)
	if err != nil {
		t.Fatalf("settle the Yggdrasil key of the peer: %v", err)
	}
	peerAddress, _ := netip.AddrFromSlice(address.AddrForKey(key.Public().(ed25519.PublicKey))[:])
	return env, peeringURL, peerAddress
}

func runningPeer(t *testing.T, env map[string]string) {
	t.Helper()
	cfg, ops := peerFrom(t, env)
	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() {
		runDone <- yacypeerassembly.RunPeer(ctx, cfg, ops, envFrom(env))
	}()
	t.Cleanup(func() {
		cancel()
		awaitShutdown(t, runDone)
	})
	awaitEveryService(t, env, ops)
}

func neighbourOf(t *testing.T, peeringURL *url.URL) yggdrasilNeighbour {
	t.Helper()
	key, err := yggdrasilkey.Settle(t.TempDir())
	if err != nil {
		t.Fatalf("settle the Yggdrasil key of the neighbour: %v", err)
	}
	node, err := yggdrasilnode.Start(
		t.Context(),
		key,
		yggdrasilnode.Peering{PeerURLs: []*url.URL{peeringURL}},
	)
	if err != nil {
		t.Fatalf("start the neighbour: %v", err)
	}
	t.Cleanup(node.Stop)
	stack, err := ipv6stack.New(node.Address(), node, node.MTU())
	if err != nil {
		t.Fatalf("make the IPv6 stack of the neighbour: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(stack.Close)
	t.Cleanup(cancel)
	go func() { _ = stack.Run(ctx) }()
	return yggdrasilNeighbour{stack: stack, address: node.Address()}
}

func (n yggdrasilNeighbour) client() *http.Client {
	return &http.Client{Transport: &http.Transport{DialContext: n.stack.DialContext}}
}

func (n yggdrasilNeighbour) awaitAnswer(t *testing.T, target string) {
	t.Helper()
	deadline := time.Now().Add(overlayDeadline)
	for time.Now().Before(deadline) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
		if err != nil {
			t.Fatalf("build the request: %v", err)
		}
		attemptCtx, cancel := context.WithTimeout(t.Context(), startDeadline)
		response, err := n.client().Do(request.WithContext(attemptCtx))
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			cancel()
			return
		}
		cancel()
		time.Sleep(startPollPause)
	}
	t.Fatalf("%s never answered over Yggdrasil", target)
}

func (n yggdrasilNeighbour) serve(t *testing.T, handler http.Handler) {
	t.Helper()
	listener, err := n.stack.Listen(neighbourSeedlistPort)
	if err != nil {
		t.Fatalf("listen on the neighbour: %v", err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: startDeadline}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
}

func (n yggdrasilNeighbour) servedSeedlistRequests(t *testing.T) <-chan string {
	t.Helper()
	requested := make(chan string, 16)
	n.serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case requested <- r.URL.Path:
		default:
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	return requested
}

func (n yggdrasilNeighbour) greetingSeeds(t *testing.T) <-chan yacymodel.Seed {
	t.Helper()
	greetingSeeds := make(chan yacymodel.Seed, 16)
	mux := http.NewServeMux()
	mux.HandleFunc("/seed.txt", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, n.seedLine(t)+"\n")
	})
	mux.HandleFunc(yacyproto.PathHello, func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, helloFormCeiling)
		//nolint:gosec // G120: body is bounded by MaxBytesReader above.
		if err := r.ParseMultipartForm(helloFormCeiling); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		hello, err := yacyproto.ParseHelloRequest(r.Context(), r.Form)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		select {
		case greetingSeeds <- hello.Seed:
		default:
		}
		w.WriteHeader(http.StatusNotFound)
	})
	n.serve(t, mux)
	return greetingSeeds
}

func (n yggdrasilNeighbour) seedLine(t *testing.T) string {
	t.Helper()
	hash, err := yacymodel.ParseHash("neighbourAAA")
	if err != nil {
		t.Fatalf("parse the hash of the neighbour: %v", err)
	}
	name, err := yacymodel.ParsePeerName("neighbour")
	if err != nil {
		t.Fatalf("parse the name of the neighbour: %v", err)
	}
	port, err := yacymodel.ParsePort(strconv.Itoa(neighbourSeedlistPort))
	if err != nil {
		t.Fatalf("parse the port of the neighbour: %v", err)
	}
	seed := yacymodel.Seed{
		Hash:        hash,
		Name:        name,
		PeerType:    yacymodel.PeerSenior,
		PrimaryHost: yacymodel.Some(yacymodel.HostFrom(n.address)),
		Port:        yacymodel.Some(port),
	}
	return yacyproto.HelloRequest{Seed: seed}.Form().Get(yacyproto.FieldSeed)
}

func (n yggdrasilNeighbour) seedlistURL() string {
	return "http://" + net.JoinHostPort(n.address.String(), strconv.Itoa(neighbourSeedlistPort)) +
		"/seed.txt"
}

func TestRunPeerServesThePeerProtocolOnItsYggdrasilAddress(t *testing.T) {
	env, peeringURL, peerAddress := peerOnYggdrasil(t)
	runningPeer(t, env)
	neighbour := neighbourOf(t, peeringURL)

	_, peerPort, _ := net.SplitHostPort(env["YACYNODE__YACY_PEER_ADDR"])
	neighbour.awaitAnswer(t, "http://"+net.JoinHostPort(peerAddress.String(), peerPort)+"/")
}

func TestRunPeerSendsARequestForAYggdrasilAddressOverYggdrasil(t *testing.T) {
	env, peeringURL, _ := peerOnYggdrasil(t)
	neighbour := neighbourOf(t, peeringURL)
	seedlistRequests := neighbour.servedSeedlistRequests(t)
	env[yacypeerassembly.EnvSeedlistURLs] = neighbour.seedlistURL()
	runningPeer(t, env)

	select {
	case path := <-seedlistRequests:
		if path != "/seed.txt" {
			t.Errorf("neighbour served %q, want the seedlist", path)
		}
	case <-time.After(overlayDeadline):
		t.Error("the seedlist request never reached the neighbour over Yggdrasil")
	}
}

func TestRunPeerGreetsAYggdrasilPeerWithItsYggdrasilAddress(t *testing.T) {
	env, peeringURL, peerAddress := peerOnYggdrasil(t)
	delete(env, "YACY_ADVERTISE_HOST")
	neighbour := neighbourOf(t, peeringURL)
	greetingSeeds := neighbour.greetingSeeds(t)
	env[yacypeerassembly.EnvSeedlistURLs] = neighbour.seedlistURL()
	runningPeer(t, env)

	select {
	case seed := <-greetingSeeds:
		host, _ := seed.PrimaryHost.Get()
		if host != yacymodel.HostFrom(peerAddress) {
			t.Errorf(
				"the peer greeted with host %s, want its Yggdrasil address %s",
				host,
				peerAddress,
			)
		}
	case <-time.After(overlayDeadline):
		t.Error("the peer never greeted the neighbour over Yggdrasil")
	}
}
