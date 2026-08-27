package pagefetch

import "github.com/nikitakarpei/yacy-rwi-node/canonicalurl"

type FetchedPage struct {
	LandedURL        canonicalurl.CanonicalURL
	ContentType      string
	Body             []byte
	Truncated        bool
	RobotsDirectives []string
}
