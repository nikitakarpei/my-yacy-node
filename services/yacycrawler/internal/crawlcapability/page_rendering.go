package crawlcapability

type PageRendering interface {
	SourceFormat() PageContentFormat
	Format() PageContentFormat
	Render(pageURL string, body []byte) ([]byte, error)
}
