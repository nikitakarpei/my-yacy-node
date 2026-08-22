package pagefetch

import "github.com/nikitakarpei/yacy-rwi-node/canonicalurl"

type FetchedPage struct {
	FinalURL             canonicalurl.CanonicalURL
	ContentType          string
	Body                 []byte
	Truncated            bool
	RefusesIndexing      bool
	RefusesLinkDiscovery bool
}
