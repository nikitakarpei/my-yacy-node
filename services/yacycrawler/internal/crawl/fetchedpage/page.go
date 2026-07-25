// Package fetchedpage holds what a fetch returned for one URL.
package fetchedpage

type Page struct {
	FinalURL             string
	RedirectChain        []string
	ContentType          string
	Body                 []byte
	Truncated            bool
	RefusesIndexing      bool
	RefusesLinkDiscovery bool
}
