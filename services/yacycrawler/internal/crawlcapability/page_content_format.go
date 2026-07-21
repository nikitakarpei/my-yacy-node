package crawlcapability

type PageContentFormat string

const (
	PageContentFormatDocumentHTML PageContentFormat = "document-html"
	PageContentFormatReadableHTML PageContentFormat = "readable-html"
	PageContentFormatReadableText PageContentFormat = "readable-text"
	PageContentFormatMarkdown     PageContentFormat = "markdown"
)
