package pagescrape

type ScrapedPage struct {
	CanonicalURL     string
	Title            string
	Language         string
	LocalLinks       int
	ExternalLinks    int
	DocumentByteSize int
	Content          []byte
}
