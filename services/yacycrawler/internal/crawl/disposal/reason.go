// Package disposal names how a pending page visit ended, publication included.
package disposal

type Reason string

const NotDisposed Reason = ""

const (
	NotDue                Reason = "not-due"
	NotModified           Reason = "not-modified"
	AccessRefused         Reason = "access-refused"
	FetchRejected         Reason = "fetch-rejected"
	RedirectTargetInvalid Reason = "redirect-target-invalid"
	Oversized             Reason = "oversized"
	UnsupportedMediaType  Reason = "unsupported-media-type"
	UnreadableHTML        Reason = "unreadable-html"
	DeferralsExhausted    Reason = "deferrals-exhausted"
	RetriesExhausted      Reason = "retries-exhausted"
	HostPagesExhausted    Reason = "host-pages-exhausted"
)

func (reason Reason) DisposedThePage() bool {
	return reason != NotDisposed
}
