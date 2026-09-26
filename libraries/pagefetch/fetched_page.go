package pagefetch

type FetchedPage struct {
	ContentType     string
	Body            []byte
	RobotsTagValues []string
}
