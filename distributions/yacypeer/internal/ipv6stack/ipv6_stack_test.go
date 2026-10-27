package ipv6stack_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/ipv6stack"
)

const (
	linkMTU     = 1280
	servicePort = 8090
	stopWait    = 10 * time.Second
)

var (
	callerAddress = netip.MustParseAddr("200::1")
	serverAddress = netip.MustParseAddr("200::2")
)

type packetWire struct {
	inbound  <-chan []byte
	outbound chan<- []byte
	closed   <-chan struct{}
}

func (w packetWire) Read(packet []byte) (int, error) {
	select {
	case received := <-w.inbound:
		return copy(packet, received), nil
	case <-w.closed:
		return 0, io.EOF
	}
}

func (w packetWire) Write(packet []byte) (int, error) {
	sent := append([]byte(nil), packet...)
	select {
	case w.outbound <- sent:
		return len(packet), nil
	case <-w.closed:
		return 0, io.ErrClosedPipe
	}
}

type refusingWire struct {
	packetWire
	refusals chan struct{}
}

func (w refusingWire) Write([]byte) (int, error) {
	select {
	case w.refusals <- struct{}{}:
	default:
	}
	return 0, errors.New("packet refused")
}

func wiredWires(t *testing.T) (packetWire, packetWire) {
	t.Helper()
	toServer, toCaller := make(chan []byte, 64), make(chan []byte, 64)
	closed := make(chan struct{})
	t.Cleanup(func() { close(closed) })
	return packetWire{inbound: toCaller, outbound: toServer, closed: closed},
		packetWire{inbound: toServer, outbound: toCaller, closed: closed}
}

func runningStack(t *testing.T, address netip.Addr, packets io.ReadWriter) *ipv6stack.Stack {
	t.Helper()
	ipStack, err := ipv6stack.New(address, packets, linkMTU)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- ipStack.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		defer ipStack.Close()
		select {
		case err := <-runDone:
			if err != nil {
				t.Errorf("Run: %v", err)
			}
		case <-time.After(stopWait):
			t.Error("Run did not stop after cancel")
		}
	})
	return ipStack
}

func listening(t *testing.T, ipStack *ipv6stack.Stack) net.Listener {
	t.Helper()
	listener, err := ipStack.Listen(servicePort)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	return listener
}

func serverAddressPort() string {
	return netip.AddrPortFrom(serverAddress, servicePort).String()
}

func TestADialedConnectionReachesTheListenerOfTheOtherStack(t *testing.T) {
	callerWire, serverWire := wiredWires(t)
	caller := runningStack(t, callerAddress, callerWire)
	listener := listening(t, runningStack(t, serverAddress, serverWire))
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
		}
	}()

	conn, err := caller.DialContext(t.Context(), "tcp", serverAddressPort())
	if err != nil {
		t.Fatalf("DialContext: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if _, err := conn.Write([]byte("hello")); err != nil {
		t.Fatalf("write: %v", err)
	}

	served := <-accepted
	t.Cleanup(func() { _ = served.Close() })
	received := make([]byte, 5)
	if _, err := io.ReadFull(served, received); err != nil || string(received) != "hello" {
		t.Errorf("served read %q (%v), want hello", received, err)
	}
	remote, _ := netip.ParseAddrPort(served.RemoteAddr().String())
	if remote.Addr() != callerAddress {
		t.Errorf("served connection comes from %s, want %s", served.RemoteAddr(), callerAddress)
	}
}

func TestDialContextRefusesANetworkThatIsNotTCP(t *testing.T) {
	callerWire, _ := wiredWires(t)
	caller := runningStack(t, callerAddress, callerWire)

	if _, err := caller.DialContext(t.Context(), "udp", serverAddressPort()); err == nil {
		t.Error("DialContext accepted udp")
	}
}

func TestDialContextRefusesAnAddressWithoutAPort(t *testing.T) {
	callerWire, _ := wiredWires(t)
	caller := runningStack(t, callerAddress, callerWire)

	if _, err := caller.DialContext(t.Context(), "tcp", serverAddress.String()); err == nil {
		t.Error("DialContext accepted an address without a port")
	}
}

func TestDialContextFailsWhenNoPacketLeaves(t *testing.T) {
	callerWire, _ := wiredWires(t)
	wire := refusingWire{packetWire: callerWire, refusals: make(chan struct{}, 1)}
	caller := runningStack(t, callerAddress, wire)
	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		<-wire.refusals
		cancel()
	}()

	if _, err := caller.DialContext(ctx, "tcp", serverAddressPort()); err == nil {
		t.Error("DialContext connected although no packet left the stack")
	}
}

func TestListenRefusesAPortThatIsTaken(t *testing.T) {
	_, serverWire := wiredWires(t)
	server := runningStack(t, serverAddress, serverWire)
	listening(t, server)

	if _, err := server.Listen(servicePort); err == nil {
		t.Error("Listen took a port that is taken")
	}
}

func TestListenRefusesAPortOutOfRange(t *testing.T) {
	_, serverWire := wiredWires(t)
	server := runningStack(t, serverAddress, serverWire)

	if _, err := server.Listen(1 << 16); err == nil {
		t.Error("Listen took port 65536")
	}
}
