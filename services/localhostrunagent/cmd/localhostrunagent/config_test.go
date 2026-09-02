package main_test

import (
	"testing"

	localhostrunagent "github.com/nikitakarpei/yacy-rwi-node/localhostrunagent/cmd/localhostrunagent"
)

func TestConfigurationTakesTheDefaultsOfTheSidecarDeployment(t *testing.T) {
	configuration := loaded(t, map[string]string{
		"LOCALHOSTRUN_AGENT_ADDRESS": "172.20.0.3",
	})

	if configuration.NodeOrigin.String() != "http://yacy-rwi-node:8090" {
		t.Errorf("NodeOrigin = %s, want http://yacy-rwi-node:8090", configuration.NodeOrigin)
	}
	if configuration.LocalhostRunHost != "localhost.run" {
		t.Errorf("LocalhostRunHost = %q, want localhost.run", configuration.LocalhostRunHost)
	}
	if configuration.KnownHostsPath != "/state/known_hosts" {
		t.Errorf("KnownHostsPath = %q, want /state/known_hosts", configuration.KnownHostsPath)
	}
	if configuration.LeaseSocketPath != "/run/yacy/process-environment-lease.sock" {
		t.Errorf("LeaseSocketPath = %q, want the default socket path",
			configuration.LeaseSocketPath)
	}
	if configuration.IdentityFilePath != "" {
		t.Errorf("IdentityFilePath = %q, want no identity file", configuration.IdentityFilePath)
	}
}

func TestConfigurationCarriesTheAddressOfTheAgent(t *testing.T) {
	configuration := loaded(t, map[string]string{
		"LOCALHOSTRUN_AGENT_ADDRESS": "172.20.0.3",
	})

	if configuration.AgentAddress.String() != "172.20.0.3" {
		t.Errorf("AgentAddress = %s, want 172.20.0.3", configuration.AgentAddress)
	}
}

func TestConfigurationRejectsAnUnusableEnvironment(t *testing.T) {
	for name, environment := range map[string]map[string]string{
		"no agent address": {},
		"agent address that is not an address": {
			"LOCALHOSTRUN_AGENT_ADDRESS": "yacy-localhostrunagent",
		},
		"node origin that is not http": {
			"LOCALHOSTRUN_AGENT_ADDRESS": "172.20.0.3",
			"YACY_NODE_ORIGIN":           "ftp://yacy-rwi-node:8090",
		},
		"node origin without a host": {
			"LOCALHOSTRUN_AGENT_ADDRESS": "172.20.0.3",
			"YACY_NODE_ORIGIN":           "http:///Network.xml",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := localhostrunagent.LoadAgentConfiguration(environmentOf(environment))
			if err == nil {
				t.Fatalf("LoadAgentConfiguration accepted %s", name)
			}
		})
	}
}

func loaded(t *testing.T, environment map[string]string) localhostrunagent.AgentConfiguration {
	t.Helper()

	configuration, err := localhostrunagent.LoadAgentConfiguration(environmentOf(environment))
	if err != nil {
		t.Fatalf("LoadAgentConfiguration: %v", err)
	}

	return configuration
}

func environmentOf(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}
