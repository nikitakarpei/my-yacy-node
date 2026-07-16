package crawlcapability

type PageRendering interface {
	Format() PageContentFormat
	SourceFormats() []PageContentFormat
	Render(body []byte, sourceFormat PageContentFormat) ([]byte, error)
}
