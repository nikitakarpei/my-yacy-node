package yacycrawlcontract

type PageVisitResult string

const (
	ResultPublished             PageVisitResult = "published"
	ResultRedirected            PageVisitResult = "redirected"
	ResultNotModified           PageVisitResult = "not-modified"
	ResultAccessRefused         PageVisitResult = "access-refused"
	ResultFetchRejected         PageVisitResult = "fetch-rejected"
	ResultNotFound              PageVisitResult = "not-found"
	ResultRedirectTargetInvalid PageVisitResult = "redirect-target-invalid"
	ResultOversized             PageVisitResult = "oversized"
	ResultUnsupportedMediaType  PageVisitResult = "unsupported-media-type"
	ResultUnreadableHTML        PageVisitResult = "unreadable-html"
)

var PageVisitResults = []PageVisitResult{
	ResultPublished,
	ResultRedirected,
	ResultNotModified,
	ResultAccessRefused,
	ResultFetchRejected,
	ResultNotFound,
	ResultRedirectTargetInvalid,
	ResultOversized,
	ResultUnsupportedMediaType,
	ResultUnreadableHTML,
}

type PageVisitDropReason string

const (
	DropReasonTakenByAnother     PageVisitDropReason = "taken-by-another"
	DropReasonNotDue             PageVisitDropReason = "not-due"
	DropReasonHostPagesExhausted PageVisitDropReason = "host-pages-exhausted"
	DropReasonDeferralsExhausted PageVisitDropReason = "deferrals-exhausted"
	DropReasonRetriesExhausted   PageVisitDropReason = "retries-exhausted"
	DropReasonOrderForgotten     PageVisitDropReason = "order-forgotten"
)

var PageVisitDropReasons = []PageVisitDropReason{
	DropReasonTakenByAnother,
	DropReasonNotDue,
	DropReasonHostPagesExhausted,
	DropReasonDeferralsExhausted,
	DropReasonRetriesExhausted,
	DropReasonOrderForgotten,
}
