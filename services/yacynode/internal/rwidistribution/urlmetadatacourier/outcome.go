package urlmetadatacourier

type Outcome string

const (
	Accepted    Outcome = "accepted"
	Deferred    Outcome = "deferred"
	Refused     Outcome = "refused"
	Unreachable Outcome = "unreachable"
)
