package documentextraction

import (
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type Document struct {
	Title                string
	Body                 []byte
	Format               Format
	Language             string
	DiscoveredURLs       []canonicalurl.CanonicalURL
	LocalLinks           int
	ExternalLinks        int
	RefusesIndexing      bool
	RefusesLinkDiscovery bool
}
