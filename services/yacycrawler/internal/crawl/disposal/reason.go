package disposal

type Reason string

const NotDisposed Reason = ""

const (
	NotDue               Reason = "not-due"
	NotModified          Reason = "not-modified"
	Refused              Reason = "refused"
	DeferralsExhausted   Reason = "deferrals-exhausted"
	NotAPage             Reason = "not-a-page"
	FetchAbandoned       Reason = "fetch-abandoned"
	BudgetTruncated      Reason = "budget-truncated"
	Oversized            Reason = "oversized"
	UnsupportedMediaType Reason = "unsupported-media-type"
	Unextractable        Reason = "unextractable"
	UncanonicalizableURL Reason = "uncanonicalizable-url"
	IndexingRefused      Reason = "indexing-refused"
)
