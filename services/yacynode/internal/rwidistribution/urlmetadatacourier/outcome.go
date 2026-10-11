package urlmetadatacourier

type Outcome string

const (
	Unaddressable Outcome = "unaddressable"
	Accepted      Outcome = "accepted"
	Deferred      Outcome = "deferred"
	Refused       Outcome = "refused"
	Unreachable   Outcome = "unreachable"
)
