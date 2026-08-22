package pagescrape

import "github.com/nikitakarpei/yacy-rwi-node/canonicalurl"

type ScrapedPage struct {
	CanonicalURL     canonicalurl.CanonicalURL
	Title            string
	Language         string
	LocalLinks       int
	ExternalLinks    int
	DocumentByteSize int
	Content          []byte
}
