package crawlcapability

// RepresentationDerivation derives one page representation from the page's content in the
// format its feed reads, rendered by the crawl before the page reaches the derivation.
type RepresentationDerivation[R any] interface {
	Derive(page CrawledPage, content []byte) (R, error)
}
