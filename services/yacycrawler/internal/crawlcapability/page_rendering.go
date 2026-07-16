package crawlcapability

type PageRendering interface {
	SourceFormat() PageContentFormat
	Format() PageContentFormat
	Render(body []byte) ([]byte, error)
}
