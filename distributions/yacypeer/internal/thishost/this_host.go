// Package thishost reads the host that runs the peer: its network interfaces that are up, their
// classes in the system, and the address of its default route.
package thishost

import (
	"context"
	"log/slog"
	"net"
	"os"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/hostaddresses"
)

const (
	probeAddr          = "192.0.2.1:9"
	interfaceClassRoot = "/sys/class/net"
)

func Host() hostaddresses.Host {
	return hostaddresses.HostFrom(interfaces(), os.DirFS(interfaceClassRoot), outgoingIP())
}

func interfaces() []hostaddresses.Interface {
	systemInterfaces, err := net.Interfaces()
	if err != nil {
		slog.WarnContext(context.Background(), "network interfaces unread", slog.Any("error", err))
		return nil
	}
	upInterfaces := make([]hostaddresses.Interface, 0, len(systemInterfaces))
	for _, networkInterface := range systemInterfaces {
		if networkInterface.Flags&net.FlagUp == 0 {
			continue
		}
		addresses, err := networkInterface.Addrs()
		if err != nil {
			slog.WarnContext(context.Background(), "addresses of a network interface unread",
				slog.String("interface", networkInterface.Name), slog.Any("error", err),
			)
			continue
		}
		ips := make([]net.IP, 0, len(addresses))
		for _, address := range addresses {
			if network, isNetwork := address.(*net.IPNet); isNetwork {
				ips = append(ips, network.IP)
			}
		}
		upInterfaces = append(
			upInterfaces,
			hostaddresses.Interface{Name: networkInterface.Name, IPs: ips},
		)
	}
	return upInterfaces
}

func outgoingIP() net.IP {
	probe, err := (&net.Dialer{}).DialContext(context.Background(), "udp", probeAddr)
	if err != nil {
		slog.DebugContext(context.Background(), "no default route", slog.Any("error", err))
		return nil
	}
	defer closeProbe(probe)
	return probe.LocalAddr().(*net.UDPAddr).IP
}

func closeProbe(probe net.Conn) {
	if err := probe.Close(); err != nil {
		slog.DebugContext(
			context.Background(),
			"default route probe not closed",
			slog.Any("error", err),
		)
	}
}
