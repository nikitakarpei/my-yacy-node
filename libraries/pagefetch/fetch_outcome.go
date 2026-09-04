package pagefetch

import "time"

type FetchOutcome struct {
	Status       FetchStatus
	DeferFor     time.Duration
	Page         FetchedPage
	Version      PageVersion
	FailureCause error
}
