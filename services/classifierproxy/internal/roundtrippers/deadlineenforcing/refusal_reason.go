package deadlineenforcing

type RefusalReason string

const (
	WaitTooShort    RefusalReason = "wait_too_short"
	HeadersDeadline RefusalReason = "headers_deadline"
)
