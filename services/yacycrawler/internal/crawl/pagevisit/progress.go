package pagevisit

import "time"

const (
	DisposalRefused     = "refused"
	DisposalFetchFailed = "fetch-failed"

	RefusalCease = "cease"
)

type Progress interface {
	PageFetched()
	PageDisposed(reason string)
	RefusalHonored(demand string)
	FetchObserved(elapsed time.Duration)
}
