package pagefetch

type FetchedPage struct {
	ContentType      string
	Body             []byte
	RobotsDirectives []string
}
