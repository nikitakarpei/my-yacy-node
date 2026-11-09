package companionprotocol

type PairingRequest struct {
	AppVersion      string   `json:"appVersion"`
	Attestation     [][]byte `json:"attestation"`
	Proof           Proof    `json:"proof"`
	ProtocolVersion int      `json:"protocolVersion"`
}

type PairingReply struct {
	Outcome  Outcome       `json:"outcome"`
	DeviceID DeviceID       `json:"deviceID,omitempty"`
	Reason   RefusalReason  `json:"reason,omitempty"`
}

func AcceptedPairingReply(deviceID DeviceID) PairingReply {
	return PairingReply{Outcome: OutcomeAccepted, DeviceID: deviceID}
}

func RefusedPairingReply(reason RefusalReason) PairingReply {
	return PairingReply{Outcome: OutcomeRefused, Reason: reason}
}
