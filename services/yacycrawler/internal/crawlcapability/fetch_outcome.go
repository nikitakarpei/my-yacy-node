package crawlcapability

import "time"

type FetchOutcome struct {
	Status               FetchStatus
	FinalURL             string
	RedirectChain        []string
	ContentType          string
	Body                 []byte
	Truncated            bool
	RefusesIndexing      bool
	RefusesLinkDiscovery bool
	DeferFor             time.Duration
}
