package companionprotocol

type Hello struct {
	AppVersion      string `json:"appVersion"`
	ProtocolVersion int    `json:"protocolVersion"`
}

type HelloReply struct {
	Outcome Outcome       `json:"outcome"`
	Reason  RefusalReason `json:"reason,omitempty"`
}

func AcceptedHelloReply() HelloReply {
	return HelloReply{Outcome: OutcomeAccepted}
}

func RefusedHelloReply(reason RefusalReason) HelloReply {
	return HelloReply{Outcome: OutcomeRefused, Reason: reason}
}

type DeviceSettings struct {
	PartitionExponent uint `json:"partitionExponent"`
}
