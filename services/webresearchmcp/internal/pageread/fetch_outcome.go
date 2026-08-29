package pageread

type FetchOutcome string

const (
	PageFetched     FetchOutcome = "page-fetched"
	PageNotReadable FetchOutcome = "page-not-readable"
	FetchUnfinished FetchOutcome = "fetch-unfinished"
	FetchNotNeeded  FetchOutcome = "fetch-not-needed"
)
