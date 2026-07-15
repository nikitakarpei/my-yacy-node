package crawlcapability

// RepresentationDerivation derives one page representation from a crawled page,
// rendering whatever content it needs from rendered rather than parsing the page itself.
type RepresentationDerivation[R any] interface {
	Name() string
	Accepts(format PageContentFormat) bool
	Derive(page CrawledPage, rendered *RenderedContent) (R, error)
}
