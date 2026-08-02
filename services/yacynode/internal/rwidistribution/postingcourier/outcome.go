package postingcourier

type Outcome string

const (
	Unaddressable Outcome = "unaddressable"
	Unreachable   Outcome = "unreachable"
	Accepted      Outcome = "accepted"
	Deferred      Outcome = "deferred"
	Overloaded    Outcome = "overloaded"
	Refused       Outcome = "refused"
)
