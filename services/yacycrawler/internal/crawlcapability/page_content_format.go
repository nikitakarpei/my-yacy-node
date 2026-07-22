package crawlcapability

type PageContentFormat string

const (
	PageContentFormatDocumentHTML PageContentFormat = "document-html"
	PageContentFormatReadableHTML PageContentFormat = "readable-html"
	PageContentFormatReadableText PageContentFormat = "readable-text"
	PageContentFormatFullText     PageContentFormat = "full-text"
	PageContentFormatMarkdown     PageContentFormat = "markdown"
)
