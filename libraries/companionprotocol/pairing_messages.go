package companionprotocol

const ProtocolVersion = 1

type PairingOutcome string

const (
	OutcomeAccepted PairingOutcome = "accepted"
	OutcomeRefused  PairingOutcome = "refused"
)

type RefusalReason string

const (
	ReasonAppOutdated      RefusalReason = "app-outdated"
	ReasonPeerOutdated     RefusalReason = "peer-outdated"
	ReasonAppNotOfficial   RefusalReason = "app-not-official"
	ReasonDeviceNotTrusted RefusalReason = "device-not-trusted"
	ReasonRequestInvalid   RefusalReason = "request-invalid"
)

type PairingRequest struct {
	AppVersion      string   `json:"appVersion"`
	Attestation     [][]byte `json:"attestation"`
	Proof           Proof    `json:"proof"`
	ProtocolVersion int      `json:"protocolVersion"`
}

type PairingReply struct {
	Outcome  PairingOutcome `json:"outcome"`
	DeviceID DeviceID       `json:"deviceID,omitempty"`
	Reason   RefusalReason  `json:"reason,omitempty"`
}

func AcceptedReply(device DeviceID) PairingReply {
	return PairingReply{Outcome: OutcomeAccepted, DeviceID: device}
}

func RefusedReply(reason RefusalReason) PairingReply {
	return PairingReply{Outcome: OutcomeRefused, Reason: reason}
}
