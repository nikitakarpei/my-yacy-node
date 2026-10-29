package nodeconfiguration

import (
	"fmt"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	EnvPeerKey                  = "YACY_PEER_KEY"
	EnvPeerName                 = "YACY_PEER_NAME"
	EnvNetworkName              = "YACY_NETWORK_NAME"
	EnvAdvertiseHost            = "YACY_ADVERTISE_HOST"
	EnvAdvertiseAdditionalHosts = "YACY_ADVERTISE_ADDITIONAL_HOSTS"
	EnvAdvertisePort            = "YACY_ADVERTISE_PORT"
	EnvAdvertiseSecurePort      = "YACY_ADVERTISE_SECURE_PORT"
	EnvAcceptRemoteIndex        = "YACY_ACCEPT_REMOTE_INDEX"

	DefaultAcceptRemoteIndex = true
)

type IdentityConfig struct {
	PeerKey                  yacymodel.PeerKey
	NetworkName              string
	Name                     yacymodel.Optional[yacymodel.PeerName]
	AdvertiseHost            yacymodel.Optional[yacymodel.Host]
	AdvertiseAdditionalHosts []yacymodel.Host
	AdvertisePort            int
	AdvertiseSecurePort      int
	Capabilities             yacymodel.PeerCapabilities
}

func loadIdentityConfig(
	getenv func(string) string,
	serving ServingConfig,
) (IdentityConfig, error) {
	peerKey, err := requiredPeerKey(getenv)
	if err != nil {
		return IdentityConfig{}, err
	}

	name, err := statedValueOf(getenv, EnvPeerName, yacymodel.ParsePeerName)
	if err != nil {
		return IdentityConfig{}, err
	}

	host, err := statedValueOf(getenv, EnvAdvertiseHost, yacymodel.ParseHost)
	if err != nil {
		return IdentityConfig{}, err
	}

	additionalHosts, err := advertisedAdditionalHosts(getenv)
	if err != nil {
		return IdentityConfig{}, err
	}

	port, err := envconfig.PositiveInt(getenv, EnvAdvertisePort, serving.PeerPort)
	if err != nil {
		return IdentityConfig{}, err
	}

	securePort, err := envconfig.PositiveInt(getenv, EnvAdvertiseSecurePort, serving.SecurePeerPort)
	if err != nil {
		return IdentityConfig{}, err
	}

	capabilities, err := peerCapabilities(getenv)
	if err != nil {
		return IdentityConfig{}, err
	}

	return IdentityConfig{
		PeerKey: peerKey,
		NetworkName: envconfig.String(
			getenv,
			EnvNetworkName,
			yacyproto.DefaultNetwork,
		),
		Name:                     name,
		AdvertiseHost:            host,
		AdvertiseAdditionalHosts: additionalHosts,
		AdvertisePort:            port,
		AdvertiseSecurePort:      securePort,
		Capabilities:             capabilities,
	}, nil
}

func requiredPeerKey(getenv func(string) string) (yacymodel.PeerKey, error) {
	stated, err := envconfig.Required(getenv, EnvPeerKey)
	if err != nil {
		return yacymodel.PeerKey{}, err
	}

	peerKey, err := yacymodel.ParsePeerKey(stated)
	if err != nil {
		return yacymodel.PeerKey{}, fmt.Errorf("%s: %w", EnvPeerKey, err)
	}

	return peerKey, nil
}

func statedValueOf[T any](
	getenv func(string) string,
	key string,
	parse func(string) (T, error),
) (yacymodel.Optional[T], error) {
	stated := strings.TrimSpace(getenv(key))
	if stated == "" {
		return yacymodel.None[T](), nil
	}

	value, err := parse(stated)
	if err != nil {
		return yacymodel.None[T](), fmt.Errorf("%s: %w", key, err)
	}

	return yacymodel.Some(value), nil
}

func advertisedAdditionalHosts(getenv func(string) string) ([]yacymodel.Host, error) {
	var hosts []yacymodel.Host
	for _, stated := range envconfig.List(getenv, EnvAdvertiseAdditionalHosts) {
		host, err := yacymodel.ParseHost(stated)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", EnvAdvertiseAdditionalHosts, err)
		}
		hosts = append(hosts, host)
	}

	return hosts, nil
}

func peerCapabilities(getenv func(string) string) (yacymodel.PeerCapabilities, error) {
	acceptRemoteIndex, err := envconfig.Bool(getenv, EnvAcceptRemoteIndex, DefaultAcceptRemoteIndex)
	if err != nil {
		return yacymodel.PeerCapabilities{}, err
	}

	return yacymodel.PeerCapabilities{
		DirectConnect:     true,
		AcceptRemoteIndex: acceptRemoteIndex,
		SSLAvailable:      true,
	}, nil
}
