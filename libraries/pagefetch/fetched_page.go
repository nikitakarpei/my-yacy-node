package pagefetch

type FetchedPage struct {
	ContentType string
	Body        []byte
	// TECHDEBT: vocabulary — RobotsDirectives holds the X-Robots-Tag values, which robotsmeta/httpheader names robotsTagValues
	RobotsDirectives []string
}
