// Package disposal names how a pending visit ended, publication included.
package disposal

type Reason string

const NotDisposed Reason = ""

const (
	NotDue               Reason = "not-due"
	NotModified          Reason = "not-modified"
	AccessRefused        Reason = "access-refused"
	FetchRejected        Reason = "fetch-rejected"
	LandedURLInvalid     Reason = "landed-url-invalid"
	Oversized            Reason = "oversized"
	UnsupportedMediaType Reason = "unsupported-media-type"
	IndexingRefused      Reason = "indexing-refused"
	DeferralsExhausted   Reason = "deferrals-exhausted"
	RetriesExhausted     Reason = "retries-exhausted"
	HostPagesExhausted   Reason = "host-pages-exhausted"
)

func (reason Reason) Disposes() bool {
	return reason != NotDisposed
}
