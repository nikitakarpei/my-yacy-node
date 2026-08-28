package pagefetch

type FetchStatus int

const (
	FetchSucceeded FetchStatus = iota
	FetchNotModified
	FetchAccessRefused
	FetchDeferred
	FetchRejected
	FetchLandedURLInvalid
	FetchOversized
	FetchFailed
)
