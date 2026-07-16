package crawlcapability

// RepresentationDerivation derives one page representation from a crawled page,
// rendering whatever content it needs through render rather than parsing the page itself.
type RepresentationDerivation[R any] interface {
	Accepts(format PageContentFormat) bool
	Derive(page CrawledPage, render RenderContent) (R, error)
}
