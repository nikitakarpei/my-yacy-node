package pagefetch

type FetchStatus int

const (
	FetchSucceeded FetchStatus = iota
	FetchNotModified
	FetchRedirected
	FetchAccessRefused
	FetchDeferred
	FetchRejected
	FetchRedirectTargetInvalid
	FetchOversized
	FetchFailed
)
