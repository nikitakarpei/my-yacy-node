// Package ipv6stack runs a TCP/IP stack in the process over a stream of raw IPv6 packets, so the
// peer can listen and dial on a network that gives it packets but no host interface. The stack
// is the gVisor network stack.
package ipv6stack

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/netip"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
)

const (
	interfaceID    = 1
	queuedPackets  = 1024
	streamNetwork  = "tcp"
	streamNetwork6 = "tcp6"
)

var (
	errNotAStreamNetwork = errors.New("the IPv6 stack dials tcp only")
	errPortOutOfRange    = errors.New("port out of range")
)

type Stack struct {
	stack   *stack.Stack
	link    *channel.Endpoint
	packets io.ReadWriter
	address netip.Addr
	mtu     uint32
}

func New(address netip.Addr, packets io.ReadWriter, mtu uint32) (*Stack, error) {
	ipStack := stack.New(stack.Options{
		NetworkProtocols:   []stack.NetworkProtocolFactory{ipv6.NewProtocol},
		TransportProtocols: []stack.TransportProtocolFactory{tcp.NewProtocol, icmp.NewProtocol6},
		HandleLocal:        true,
	})
	link := channel.New(queuedPackets, mtu, "")
	if err := ipStack.CreateNIC(interfaceID, link); err != nil {
		return nil, fmt.Errorf("make the IPv6 interface: %s", err)
	}
	if err := ipStack.AddProtocolAddress(interfaceID, tcpip.ProtocolAddress{
		Protocol:          ipv6.ProtocolNumber,
		AddressWithPrefix: tcpip.AddrFrom16(address.As16()).WithPrefix(),
	}, stack.AddressProperties{}); err != nil {
		return nil, fmt.Errorf("give the IPv6 interface the address %s: %s", address, err)
	}
	ipStack.AddRoute(tcpip.Route{Destination: header.IPv6EmptySubnet, NIC: interfaceID})
	return &Stack{stack: ipStack, link: link, packets: packets, address: address, mtu: mtu}, nil
}

func (s *Stack) Listen(port int) (net.Listener, error) {
	if port < 0 || port > math.MaxUint16 {
		return nil, fmt.Errorf("listen on [%s]:%d: %w", s.address, port, errPortOutOfRange)
	}
	listener, err := gonet.ListenTCP(
		s.stack,
		s.fullAddressOf(netip.AddrPortFrom(s.address, uint16(port))),
		ipv6.ProtocolNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("listen on [%s]:%d: %w", s.address, port, err)
	}
	return listener, nil
}

func (s *Stack) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if network != streamNetwork && network != streamNetwork6 {
		return nil, fmt.Errorf("%w: %q", errNotAStreamNetwork, network)
	}
	destination, err := netip.ParseAddrPort(address)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", address, err)
	}
	conn, err := gonet.DialContextTCP(
		ctx,
		s.stack,
		s.fullAddressOf(destination),
		ipv6.ProtocolNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", address, err)
	}
	return conn, nil
}

func (s *Stack) fullAddressOf(addressPort netip.AddrPort) tcpip.FullAddress {
	return tcpip.FullAddress{
		NIC:  interfaceID,
		Addr: tcpip.AddrFrom16(addressPort.Addr().As16()),
		Port: addressPort.Port(),
	}
}

func (s *Stack) Run(ctx context.Context) error {
	go s.carryInboundPackets(ctx)
	s.carryOutboundPackets(ctx)
	return nil
}

func (s *Stack) Close() {
	s.stack.Close()
}

func (s *Stack) carryInboundPackets(ctx context.Context) {
	packet := make([]byte, s.mtu)
	for {
		size, err := s.packets.Read(packet)
		if err != nil {
			slog.DebugContext(ctx, "IPv6 stack stopped reading packets", slog.Any("reason", err))
			return
		}
		buffered := stack.NewPacketBuffer(stack.PacketBufferOptions{
			Payload: buffer.MakeWithData(packet[:size]),
		})
		s.link.InjectInbound(ipv6.ProtocolNumber, buffered)
		buffered.DecRef()
	}
}

func (s *Stack) carryOutboundPackets(ctx context.Context) {
	for {
		buffered := s.link.ReadContext(ctx)
		if buffered == nil {
			return
		}
		view := buffered.ToView()
		buffered.DecRef()
		if _, err := s.packets.Write(view.AsSlice()); err != nil {
			slog.WarnContext(ctx, "IPv6 packet dropped", slog.Any("reason", err))
		}
		view.Release()
	}
}
