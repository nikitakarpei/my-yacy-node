package nodeconfiguration

import "fmt"

func requireOnePostingSource(identity IdentityConfig, pageOfferIntake PageOfferIntakeConfig) error {
	acceptsRemoteIndex := identity.Capabilities.AcceptRemoteIndex
	if acceptsRemoteIndex == pageOfferIntake.Enabled() {
		return fmt.Errorf(
			"%s=%t with %s=%q: exactly one source of postings must be enabled",
			EnvAcceptRemoteIndex, acceptsRemoteIndex,
			EnvPageOfferNATSURL, pageOfferIntake.PageOfferNATSURL,
		)
	}

	return nil
}
