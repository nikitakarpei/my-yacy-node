package nodeconfiguration

import "fmt"

func requireOnePostingSource(identity IdentityConfig, pageRWIIntake PageRWIIntakeConfig) error {
	acceptsRemoteIndex := identity.Capabilities.AcceptRemoteIndex
	if acceptsRemoteIndex == pageRWIIntake.Enabled() {
		return fmt.Errorf(
			"%s=%t with %s=%q: exactly one source of postings must be enabled",
			EnvAcceptRemoteIndex, acceptsRemoteIndex,
			EnvPageRWINATSURL, pageRWIIntake.PageRWINATSURL,
		)
	}

	return nil
}
