// Package disposal names why a page reached a terminal outcome without publication.
package disposal

type Reason string

const NotDisposed Reason = ""

const (
	NotDue               Reason = "not-due"
	NotModified          Reason = "not-modified"
	Refused              Reason = "refused"
	NotAPage             Reason = "not-a-page"
	Oversized            Reason = "oversized"
	UnsupportedMediaType Reason = "unsupported-media-type"
	IndexingRefused      Reason = "indexing-refused"
)
