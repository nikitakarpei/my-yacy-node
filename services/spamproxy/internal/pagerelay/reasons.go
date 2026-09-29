package pagerelay

import "net/http"

type NonPageReason string

const (
	HeadRequest NonPageReason = "head_request"
	Not2xx      NonPageReason = "not_2xx"
	NotHTML     NonPageReason = "not_html"
)

type RefusalReason string

const (
	WaitTooShort          RefusalReason = "wait_too_short"
	PageReadTimeout       RefusalReason = "page_read_timeout"
	ResponseHeaderTimeout RefusalReason = "response_header_timeout"
	UndecodableEncoding   RefusalReason = "undecodable_encoding"
	UndecodableBody       RefusalReason = "undecodable_body"
	TooManyAtOnce         RefusalReason = "too_many_at_once"
	UnassessedPage        RefusalReason = "unassessed_page"
)

var refusalStatuses = map[RefusalReason]int{
	WaitTooShort:          http.StatusGatewayTimeout,
	PageReadTimeout:       http.StatusGatewayTimeout,
	ResponseHeaderTimeout: http.StatusGatewayTimeout,
	UndecodableEncoding:   http.StatusBadGateway,
	UndecodableBody:       http.StatusBadGateway,
	TooManyAtOnce:         http.StatusServiceUnavailable,
	UnassessedPage:        http.StatusServiceUnavailable,
}

func (r RefusalReason) isRetryable() bool {
	return r == TooManyAtOnce || r == UnassessedPage
}
