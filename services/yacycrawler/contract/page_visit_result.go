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
	ResultUnreadableHTML,
}

type PageVisitDropReason string

const (
	DropReasonNotDue             PageVisitDropReason = "not-due"
	DropReasonHostPagesExhausted PageVisitDropReason = "host-pages-exhausted"
	DropReasonDeferralsExhausted PageVisitDropReason = "deferrals-exhausted"
	DropReasonRetriesExhausted   PageVisitDropReason = "retries-exhausted"
	DropReasonCrawlCancelled     PageVisitDropReason = "crawl-cancelled"
)

var PageVisitDropReasons = []PageVisitDropReason{
	DropReasonNotDue,
	DropReasonHostPagesExhausted,
	DropReasonDeferralsExhausted,
	DropReasonRetriesExhausted,
	DropReasonCrawlCancelled,
}
