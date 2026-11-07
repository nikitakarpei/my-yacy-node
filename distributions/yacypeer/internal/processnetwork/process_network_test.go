package processnetwork_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/processnetwork"
)

func TestDialReachesTheListenerAtTheHost(t *testing.T) {
	network := processnetwork.New()
	listener := network.ListenerAt("crawler.internal")
	defer func() { _ = listener.Close() }()
	if listener.Addr().String() != "crawler.internal" {
		t.Errorf("address = %s, want crawler.internal", listener.Addr())
	}
	server := httptest.NewUnstartedServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "crawls")
		}),
	)
	server.Listener = listener
	server.Start()
	defer server.Close()

	client := &http.Client{Transport: &http.Transport{DialContext: network.DialContext}}
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://crawler.internal/crawls",
		nil,
	)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(response.Body)
	if string(body) != "crawls" {
		t.Errorf("body = %q, want %q", body, "crawls")
	}
}

func TestDialReachesAnAttachedHost(t *testing.T) {
	network := processnetwork.New()
	server, attached := net.Pipe()
	defer func() { _ = server.Close() }()
	network.Attach("broker.internal", func() (net.Conn, error) { return attached, nil })

	connection, err := network.DialContext(context.Background(), "tcp", "broker.internal:4222")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if connection != attached {
		t.Errorf("dial gave %v, want the attached connection", connection)
	}
}

func TestDialReachesOtherAddressesOnTheHostNetwork(t *testing.T) {
	hostListener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = hostListener.Close() }()
	accepted := make(chan struct{})
	go func() {
		if connection, err := hostListener.Accept(); err == nil {
			_ = connection.Close()
			close(accepted)
		}
	}()

	connection, err := processnetwork.New().
		DialContext(context.Background(), "tcp", hostListener.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	_ = connection.Close()
	<-accepted
}

func TestTakingAHostTwicePanics(t *testing.T) {
	network := processnetwork.New()
	network.ListenerAt("crawler.internal")

	defer func() {
		if recover() == nil {
			t.Error("a second part at the same host was accepted")
		}
	}()
	network.Attach("crawler.internal", func() (net.Conn, error) { return nil, nil })
}

func TestAClosedListenerRefusesDials(t *testing.T) {
	network := processnetwork.New()
	listener := network.ListenerAt("crawler.internal")
	_ = listener.Close()

	if _, err := network.DialContext(
		context.Background(),
		"tcp",
		"crawler.internal:80",
	); !errors.Is(
		err,
		net.ErrClosed,
	) {
		t.Errorf("dial error = %v, want %v", err, net.ErrClosed)
	}
	if _, err := listener.Accept(); !errors.Is(err, net.ErrClosed) {
		t.Errorf("accept error = %v, want %v", err, net.ErrClosed)
	}
}

func TestADialEndsWithItsContext(t *testing.T) {
	network := processnetwork.New()
	listener := network.ListenerAt("crawler.internal")
	defer func() { _ = listener.Close() }()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := network.DialContext(
		ctx,
		"tcp",
		"crawler.internal",
	); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Errorf("dial error = %v, want %v", err, context.Canceled)
	}
}
