package documentextraction

type Format string

const (
	FormatDocumentHTML  Format = "document-html"
	FormatReadableHTML  Format = "readable-html"
	FormatReadableText  Format = "readable-text"
	FormatAuxiliaryText Format = "auxiliary-text"
	FormatFullText      Format = "full-text"
	FormatMarkdown      Format = "markdown"
)
