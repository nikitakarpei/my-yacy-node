package crawlcapability

type PageDerivation interface {
	SourceFormat() PageContentFormat
	TargetFormat() PageContentFormat
	Derive(pageURL string, body []byte) ([]byte, error)
}
