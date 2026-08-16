package nodeconfiguration

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	EnvPeerHash      = "YACY_PEER_HASH"
	EnvPeerName      = "YACY_PEER_NAME"
	EnvNetworkName   = "YACY_NETWORK_NAME"
	EnvAdvertiseHost = "YACY_ADVERTISE_HOST"
	EnvAdvertisePort = "YACY_ADVERTISE_PORT"
)

type IdentityConfig struct {
	Hash          yacymodel.Hash
	NetworkName   string
	Name          yacymodel.PeerName
	AdvertiseHost string
	AdvertisePort int
	Flags         yacymodel.PeerCapabilities
}

func loadIdentityConfig(
	getenv func(string) string,
	listenAddr string,
	announcing bool,
) (IdentityConfig, error) {
	hash, err := yacymodel.ParseHash(strings.TrimSpace(getenv(EnvPeerHash)))
	if err != nil {
		return IdentityConfig{}, fmt.Errorf("%s: %w", EnvPeerHash, err)
	}

	name, err := peerName(getenv)
	if err != nil {
		return IdentityConfig{}, err
	}

	host, err := advertiseHost(getenv, announcing)
	if err != nil {
		return IdentityConfig{}, err
	}

	port, err := advertisePort(getenv, listenAddr)
	if err != nil {
		return IdentityConfig{}, err
	}

	return IdentityConfig{
		Hash:          hash,
		NetworkName:   envconfig.String(getenv, EnvNetworkName, yacyproto.DefaultNetwork),
		Name:          name,
		AdvertiseHost: host,
		AdvertisePort: port,
		Flags:         seniorFlags(),
	}, nil
}

func peerName(getenv func(string) string) (yacymodel.PeerName, error) {
	raw, err := envconfig.Required(getenv, EnvPeerName)
	if err != nil {
		return yacymodel.PeerName{}, err
	}

	name, err := yacymodel.ParsePeerName(raw)
	if err != nil {
		return yacymodel.PeerName{}, fmt.Errorf("%s: %w", EnvPeerName, err)
	}

	return name, nil
}

func advertiseHost(getenv func(string) string, announcing bool) (string, error) {
	host := strings.TrimSpace(getenv(EnvAdvertiseHost))
	if host == "" && announcing {
		return "", fmt.Errorf("%s: must be set when announcing to the network", EnvAdvertiseHost)
	}

	return host, nil
}

func advertisePort(getenv func(string) string, listenAddr string) (int, error) {
	listenPort, err := listenPortOf(listenAddr)
	if err != nil {
		return 0, err
	}

	return envconfig.PositiveInt(getenv, EnvAdvertisePort, listenPort)
}

func listenPortOf(listenAddr string) (int, error) {
	_, portPart, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", EnvPeerAddr, err)
	}

	port, err := strconv.Atoi(portPart)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", EnvPeerAddr, err)
	}
	if port <= 0 {
		return 0, fmt.Errorf("%s: must carry a positive port", EnvPeerAddr)
	}

	return port, nil
}

func seniorFlags() yacymodel.PeerCapabilities {
	return yacymodel.PeerCapabilities{
		DirectConnect:     true,
		AcceptRemoteIndex: true,
	}
}
