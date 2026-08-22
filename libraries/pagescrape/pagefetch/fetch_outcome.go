package pagefetch

import "time"

type FetchOutcome struct {
	Status        FetchStatus
	DeferFor      time.Duration
	Page          FetchedPage
	RedirectChain []string
	Version       PageVersion
}
