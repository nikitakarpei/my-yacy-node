package pywb

import "fmt"

type capture struct {
	URLKey      string
	Timestamp   string
	OriginalURL string
}

type captureSelection struct {
	captures     []capture
	capturesRead int
	hasMorePages bool
}

type newestCaptureSelection struct {
	captureSelection           captureSelection
	newestCaptureOfCurrentPage capture
	hasCurrentPage             bool
}

func (s *newestCaptureSelection) add(captured capture, pageLimit int) (bool, error) {
	s.captureSelection.capturesRead++
	if !s.hasCurrentPage {
		s.newestCaptureOfCurrentPage = captured
		s.hasCurrentPage = true
		return false, nil
	}
	if captured.URLKey < s.newestCaptureOfCurrentPage.URLKey {
		return false, fmt.Errorf(
			"read cdx row: url key %q follows %q",
			captured.URLKey,
			s.newestCaptureOfCurrentPage.URLKey,
		)
	}
	if captured.URLKey == s.newestCaptureOfCurrentPage.URLKey {
		if captured.Timestamp > s.newestCaptureOfCurrentPage.Timestamp {
			s.newestCaptureOfCurrentPage = captured
		}
		return false, nil
	}
	s.selectCurrentPage()
	if pageLimit > 0 && len(s.captureSelection.captures) == pageLimit {
		s.captureSelection.hasMorePages = true
		return true, nil
	}
	s.newestCaptureOfCurrentPage = captured
	s.hasCurrentPage = true
	return false, nil
}

func (s *newestCaptureSelection) complete() captureSelection {
	s.selectCurrentPage()
	return s.captureSelection
}

func (s *newestCaptureSelection) selectCurrentPage() {
	if s.hasCurrentPage {
		s.captureSelection.captures = append(
			s.captureSelection.captures,
			s.newestCaptureOfCurrentPage,
		)
		s.hasCurrentPage = false
	}
}
