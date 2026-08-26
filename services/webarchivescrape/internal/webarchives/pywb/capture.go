package pywb

type Capture struct {
	URLKey      string
	Timestamp   string
	OriginalURL string
}

type Query struct {
	URL        string
	MatchType  string
	MediaType  string
	StatusCode int
	From       string
	To         string
	Limit      int
}

func NewestCapturesOf(captures []Capture) []Capture {
	newestCaptures := []Capture{}
	positionOfURLKey := map[string]int{}
	for _, capture := range captures {
		position, seen := positionOfURLKey[capture.URLKey]
		if !seen {
			positionOfURLKey[capture.URLKey] = len(newestCaptures)
			newestCaptures = append(newestCaptures, capture)
			continue
		}
		if capture.Timestamp > newestCaptures[position].Timestamp {
			newestCaptures[position] = capture
		}
	}
	return newestCaptures
}
