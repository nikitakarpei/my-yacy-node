package main

import (
	"fmt"
	"net/netip"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvNodeOrigin       = "YACY_NODE_ORIGIN"
	EnvLeaseSocketPath  = "PROCESS_ENVIRONMENT_LEASE_SOCKET"
	EnvLocalhostRunHost = "LOCALHOST_RUN_HOST"
	EnvIdentityFilePath = "LOCALHOST_RUN_IDENTITY_FILE"
	EnvKnownHostsPath   = "LOCALHOST_RUN_KNOWN_HOSTS"
	EnvAgentAddress     = "LOCALHOSTRUN_AGENT_ADDRESS"

	DefaultNodeOrigin       = "http://yacy-rwi-node:8090"
	DefaultLeaseSocketPath  = "/run/yacy/process-environment-lease.sock"
	DefaultLocalhostRunHost = "localhost.run"
	DefaultKnownHostsPath   = "/state/known_hosts"
)

type AgentConfiguration struct {
	NodeOrigin       *url.URL
	LeaseSocketPath  string
	LocalhostRunHost string
	IdentityFilePath string
	KnownHostsPath   string
	AgentAddress     netip.Addr
}

func LoadAgentConfiguration(getenv func(string) string) (AgentConfiguration, error) {
	nodeOrigin, err := url.Parse(envconfig.String(getenv, EnvNodeOrigin, DefaultNodeOrigin))
	if err != nil {
		return AgentConfiguration{}, fmt.Errorf("%s: %w", EnvNodeOrigin, err)
	}
	if nodeOrigin.Scheme != "http" || nodeOrigin.Host == "" {
		return AgentConfiguration{}, fmt.Errorf("%s: must be an http origin with a host",
			EnvNodeOrigin)
	}

	agentAddress, err := requiredAddress(getenv, EnvAgentAddress)
	if err != nil {
		return AgentConfiguration{}, err
	}

	return AgentConfiguration{
		NodeOrigin:       nodeOrigin,
		LeaseSocketPath:  envconfig.String(getenv, EnvLeaseSocketPath, DefaultLeaseSocketPath),
		LocalhostRunHost: envconfig.String(getenv, EnvLocalhostRunHost, DefaultLocalhostRunHost),
		IdentityFilePath: envconfig.String(getenv, EnvIdentityFilePath, ""),
		KnownHostsPath:   envconfig.String(getenv, EnvKnownHostsPath, DefaultKnownHostsPath),
		AgentAddress:     agentAddress,
	}, nil
}

func requiredAddress(getenv func(string) string, name string) (netip.Addr, error) {
	raw, err := envconfig.Required(getenv, name)
	if err != nil {
		return netip.Addr{}, err
	}

	address, err := netip.ParseAddr(raw)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%s: %w", name, err)
	}

	return address, nil
}
