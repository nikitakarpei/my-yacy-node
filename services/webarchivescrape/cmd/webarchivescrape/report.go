package main

import (
	"fmt"
	"io"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	webarchivespywb "github.com/nikitakarpei/yacy-rwi-node/webarchivescrape/internal/webarchives/pywb"
)

func writePublishedPage(published io.Writer, replayURL canonicalurl.CanonicalURL) error {
	if _, err := fmt.Fprintln(published, replayURL.String()); err != nil {
		return fmt.Errorf("write published page %s: %w", replayURL, err)
	}
	return nil
}

func reportSelectedPages(report io.Writer, newestReplayURLs webarchivespywb.NewestReplayURLs) {
	_, _ = fmt.Fprintf(
		report,
		"read %d captures, selected %d pages\n",
		newestReplayURLs.CapturesRead,
		len(newestReplayURLs.ReplayURLs),
	)
	if newestReplayURLs.HasMorePages {
		_, _ = fmt.Fprintln(report, "the page limit was spent; the archive holds more pages")
	}
}
