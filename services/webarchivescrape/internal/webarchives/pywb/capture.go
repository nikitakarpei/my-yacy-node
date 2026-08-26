package pywb

import "fmt"

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
}

type NewestCaptures struct {
	Captures     []Capture
	CapturesRead int
	HasMorePages bool
}

type newestCaptureSelection struct {
	newestCaptures             NewestCaptures
	newestCaptureOfCurrentPage Capture
	hasCurrentPage             bool
}

func (s *newestCaptureSelection) add(capture Capture, pageLimit int) (bool, error) {
	s.newestCaptures.CapturesRead++
	if !s.hasCurrentPage {
		s.newestCaptureOfCurrentPage = capture
		s.hasCurrentPage = true
		return false, nil
	}
	if capture.URLKey < s.newestCaptureOfCurrentPage.URLKey {
		return false, fmt.Errorf(
			"read cdx row: url key %q follows %q",
			capture.URLKey,
			s.newestCaptureOfCurrentPage.URLKey,
		)
	}
	if capture.URLKey == s.newestCaptureOfCurrentPage.URLKey {
		if capture.Timestamp > s.newestCaptureOfCurrentPage.Timestamp {
			s.newestCaptureOfCurrentPage = capture
		}
		return false, nil
	}
	s.selectCurrentPage()
	if pageLimit > 0 && len(s.newestCaptures.Captures) == pageLimit {
		s.newestCaptures.HasMorePages = true
		return true, nil
	}
	s.newestCaptureOfCurrentPage = capture
	s.hasCurrentPage = true
	return false, nil
}

func (s *newestCaptureSelection) complete() NewestCaptures {
	s.selectCurrentPage()
	return s.newestCaptures
}

func (s *newestCaptureSelection) selectCurrentPage() {
	if s.hasCurrentPage {
		s.newestCaptures.Captures = append(
			s.newestCaptures.Captures,
			s.newestCaptureOfCurrentPage,
		)
		s.hasCurrentPage = false
	}
}
