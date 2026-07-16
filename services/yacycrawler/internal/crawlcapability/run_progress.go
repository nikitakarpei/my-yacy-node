package crawlcapability

import "time"

const (
	DisposalRefused              = "refused"
	DisposalNoIndex              = "noindex"
	DisposalUnsupportedMediaType = "unsupported-media-type"
	DisposalOversized            = "oversized"
	DisposalUnextractable        = "unextractable"
	DisposalUnrepresentable      = "unrepresentable"
	DisposalFetchFailed          = "fetch-failed"
	DisposalBudgetTruncated      = "budget-truncated"
	DisposalContainerOverflow    = "container-overflow"
)

const (
	RefusalCeased   = "ceased"
	RefusalDeferred = "deferred"
)

type RunProgress interface {
	OrderReceived()
	OrderCompleted()
	OrderRedelivered()
	PageFetched()
	PagePublished(representation string)
	PageDisposed(reason string)
	RefusalHonored(kind string)
	PublicationWaited()
	FetchObserved(elapsed time.Duration)
	BudgetExhausted()
}
