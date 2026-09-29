package requestrelay

type UpstreamResponseFailure string

const (
	NoResponse           UpstreamResponseFailure = "no_response"
	BodyPrefixReadFailed UpstreamResponseFailure = "body_read_failed"
)
