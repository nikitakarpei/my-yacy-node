package pagefetch

type FetchedPage struct {
	FinalURL             string
	ContentType          string
	Body                 []byte
	Truncated            bool
	RefusesIndexing      bool
	RefusesLinkDiscovery bool
}
