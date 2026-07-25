package pagevisit

type FetchStatus int

const (
	FetchSucceeded FetchStatus = iota
	FetchCeased
	FetchDeferred
	FetchNotAPage
	FetchFailed
)
