package rwidistribution

type urlMetadataOutcome string

const (
	urlMetadataAccepted    urlMetadataOutcome = "accepted"
	urlMetadataDeferred    urlMetadataOutcome = "deferred"
	urlMetadataRefused     urlMetadataOutcome = "refused"
	urlMetadataUnreachable urlMetadataOutcome = "unreachable"
	urlMetadataUnavailable urlMetadataOutcome = "unavailable"
)
