package companionprotocol

const ProtocolVersion = 1

type Outcome string

const (
	OutcomeAccepted Outcome = "accepted"
	OutcomeRefused  Outcome = "refused"
)

type RefusalReason string

const (
	ReasonAppOutdated      RefusalReason = "app-outdated"
	ReasonPeerOutdated     RefusalReason = "peer-outdated"
	ReasonAppNotOfficial   RefusalReason = "app-not-official"
	ReasonDeviceNotTrusted RefusalReason = "device-not-trusted"
	ReasonRequestInvalid   RefusalReason = "request-invalid"
)
