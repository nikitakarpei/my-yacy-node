package companionprotocol

const ProtocolVersion = 1

type Outcome string

const (
	OutcomeAccepted Outcome = "accepted"
	OutcomeRefused  Outcome = "refused"
)

type Reason string

const (
	ReasonAppOutdated      Reason = "app-outdated"
	ReasonPeerOutdated     Reason = "peer-outdated"
	ReasonAppNotOfficial   Reason = "app-not-official"
	ReasonDeviceNotTrusted Reason = "device-not-trusted"
	ReasonRequestInvalid   Reason = "request-invalid"
)

type PairingRequest struct {
	AppVersion  string   `json:"appVersion"`
	Attestation [][]byte `json:"attestation"`
	Proof       []byte   `json:"proof"`
	Protocol    int      `json:"protocol"`
}

type PairingReply struct {
	Outcome Outcome  `json:"outcome"`
	Device  DeviceID `json:"device,omitempty"`
	Reason  Reason   `json:"reason,omitempty"`
}

func AcceptedReply(device DeviceID) PairingReply {
	return PairingReply{Outcome: OutcomeAccepted, Device: device}
}

func RefusedReply(reason Reason) PairingReply {
	return PairingReply{Outcome: OutcomeRefused, Reason: reason}
}

type Settings struct {
	PartitionExponent uint `json:"partitionExponent"`
}

type AppReport struct {
	AppVersion string `json:"appVersion"`
}
