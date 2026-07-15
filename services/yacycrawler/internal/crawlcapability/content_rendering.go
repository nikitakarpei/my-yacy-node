package crawlcapability

type ContentRendering interface {
	Format() PageContentFormat
	SourceFormats() []PageContentFormat
	Render(body []byte, sourceFormat PageContentFormat) ([]byte, error)
}
