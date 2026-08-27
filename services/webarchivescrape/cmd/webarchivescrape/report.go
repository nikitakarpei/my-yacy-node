package main

import (
	"fmt"
	"io"

	webarchivespywb "github.com/nikitakarpei/yacy-rwi-node/webarchivescrape/internal/webarchives/pywb"
)

func writePublishedPage(published io.Writer, archivedPage webarchivespywb.ArchivedPage) error {
	if _, err := fmt.Fprintln(published, archivedPage.ReplayURL.String()); err != nil {
		return fmt.Errorf("write published page %s: %w", archivedPage.ReplayURL, err)
	}
	return nil
}

func reportSelectedPages(
	report io.Writer,
	newestArchivedPages webarchivespywb.NewestArchivedPages,
) {
	_, _ = fmt.Fprintf(
		report,
		"read %d captures, selected %d pages\n",
		newestArchivedPages.CapturesRead,
		len(newestArchivedPages.ArchivedPages),
	)
	if newestArchivedPages.HasMorePages {
		_, _ = fmt.Fprintln(report, "the page limit was spent; the archive holds more pages")
	}
}
