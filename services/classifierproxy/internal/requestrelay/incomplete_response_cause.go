package requestrelay

type IncompleteResponseCause string

const (
	BodyRestReadFailed  IncompleteResponseCause = "body_read_failed"
	RelayIdleTimeout    IncompleteResponseCause = "relay_idle_timeout"
	ClientClosedRequest IncompleteResponseCause = "client_closed_request"
)
