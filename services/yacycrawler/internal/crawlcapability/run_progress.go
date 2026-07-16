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
	RefusalCease = "cease"
	RefusalDefer = "defer"
)

type RunProgress interface {
	OrderReceived()
	OrderCompleted()
	OrderRedelivered()
	PageFetched()
	PagePublished(representation string)
	PageDisposed(reason string)
	RefusalHonored(demand string)
	PublicationWaited()
	FetchObserved(elapsed time.Duration)
	BudgetExhausted()
}
