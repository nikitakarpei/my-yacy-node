package pagefetch

type FetchStatus int

const (
	FetchSucceeded FetchStatus = iota
	FetchNotModified
	FetchCeased
	FetchDeferred
	FetchRejected
	FetchLandedURLInvalid
	FetchFailed
)
