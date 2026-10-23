package searchdocument

const (
	SearchPath    = "/search"
	QueryParam    = "query"
	LanguageParam = "language"
	OffsetParam   = "offset"
	LimitParam    = "limit"
)

type PageMatches struct {
	Matches []PageMatch `json:"matches"`
}

type PageMatch struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}
