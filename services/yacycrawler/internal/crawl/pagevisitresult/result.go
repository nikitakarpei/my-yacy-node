// Package pagevisitresult names how a completed page visit ended: the page published or
// redirected, or why it published no page.
package pagevisitresult

type Result string

const (
	Published             Result = "published"
	Redirected            Result = "redirected"
	NotModified           Result = "not-modified"
	AccessRefused         Result = "access-refused"
	FetchRejected         Result = "fetch-rejected"
	RedirectTargetInvalid Result = "redirect-target-invalid"
	Oversized             Result = "oversized"
	UnsupportedMediaType  Result = "unsupported-media-type"
	UnreadableHTML        Result = "unreadable-html"
)
