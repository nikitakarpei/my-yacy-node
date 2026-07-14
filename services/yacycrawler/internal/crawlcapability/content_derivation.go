package crawlcapability

type ContentDerivation interface {
	Format() PageContentFormat
	SourceFormats() []PageContentFormat
	Derive(body []byte, sourceFormat PageContentFormat) ([]byte, error)
}
