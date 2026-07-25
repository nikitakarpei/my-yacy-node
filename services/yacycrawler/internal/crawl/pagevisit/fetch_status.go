package pagevisit

type FetchStatus int

const (
	FetchSucceeded FetchStatus = iota
	FetchNotModified
	FetchCeased
	FetchDeferred
	FetchNotAPage
	FetchFailed
)
