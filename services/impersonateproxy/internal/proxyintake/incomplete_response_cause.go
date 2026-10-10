package proxyintake

type IncompleteResponseCause string

const (
	PageReadFailed      IncompleteResponseCause = "page_read_failed"
	ClientClosedRequest IncompleteResponseCause = "client_closed_request"
)
