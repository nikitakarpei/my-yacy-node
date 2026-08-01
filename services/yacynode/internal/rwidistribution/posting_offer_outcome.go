package rwidistribution

type postingOfferOutcome string

const (
	postingOfferUnaddressable postingOfferOutcome = "unaddressable"
	postingOfferUnreachable   postingOfferOutcome = "unreachable"
	postingOfferAccepted      postingOfferOutcome = "accepted"
	postingOfferDeferred      postingOfferOutcome = "deferred"
	postingOfferOverloaded    postingOfferOutcome = "overloaded"
	postingOfferRefused       postingOfferOutcome = "refused"
)
