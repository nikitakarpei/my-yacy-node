package applog_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	scrapeprogressobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/scrapeprogressobservers/applog"
)

type recordedLogLine struct {
	level      slog.Level
	message    string
	attributes map[string]any
}

type recordedLogLines struct {
	slog.Handler
	lines []recordedLogLine
}

func (r *recordedLogLines) Enabled(context.Context, slog.Level) bool { return true }

func (r *recordedLogLines) Handle(_ context.Context, record slog.Record) error {
	attributes := map[string]any{}
	record.Attrs(func(attribute slog.Attr) bool {
		attributes[attribute.Key] = attribute.Value.Any()

		return true
	})
	r.lines = append(r.lines, recordedLogLine{
		level:      record.Level,
		message:    record.Message,
		attributes: attributes,
	})

	return nil
}

func TestScrapeProgressLogWritesEveryScrapeFactAtItsOperationalLevel(t *testing.T) {
	lines := &recordedLogLines{Handler: slog.Default().Handler()}
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(lines))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	ctx := context.Background()
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.test/page")
	cause := errors.New("scrape failed")
	progress := scrapeprogressobserversapplog.ScrapeProgressLog{}

	progress.ScrapeRequestInvalid(ctx)
	progress.OriginFetchFailed(ctx, "message", pageURL, cause)
	progress.OriginFetchDeferred(ctx, "message", pageURL, time.Second)
	progress.NothingToScrape(ctx, "message", pageURL)
	progress.DocumentExtractionFailed(ctx, "message", pageURL, cause)
	progress.NoIndexDerived(ctx, "message", pageURL)
	progress.URLMetadataAdmitted(ctx, "message", pageURL)
	progress.URLMetadataAdmissionBusy(ctx, "message", pageURL)
	progress.URLMetadataAdmissionFailed(ctx, "message", pageURL, cause)
	progress.PostingsAdmitted(ctx, "message", pageURL, 17)
	progress.PostingsAdmissionBusy(ctx, "message", pageURL, 11)
	progress.PostingsAdmissionFailed(ctx, "message", pageURL, 13, cause)
	progress.ScrapeRequestCompleted(ctx, "message", pageURL)

	wantLevels := []slog.Level{
		slog.LevelWarn,
		slog.LevelDebug,
		slog.LevelDebug,
		slog.LevelWarn,
		slog.LevelDebug,
		slog.LevelDebug,
		slog.LevelWarn,
		slog.LevelWarn,
		slog.LevelDebug,
		slog.LevelWarn,
		slog.LevelWarn,
		slog.LevelDebug,
	}
	if len(lines.lines) != len(wantLevels) {
		t.Fatalf("logged %d lines, want %d", len(lines.lines), len(wantLevels))
	}
	for lineIndex, wantLevel := range wantLevels {
		line := lines.lines[lineIndex]
		if line.level != wantLevel {
			t.Errorf("line %q level = %s, want %s", line.message, line.level, wantLevel)
		}
		if line.attributes["message"] != "message" {
			t.Errorf("line %q message identity = %v, want message", line.message,
				line.attributes["message"])
		}
		if line.attributes["fetchUrl"] != pageURL.String() &&
			line.attributes["pageUrl"] != pageURL.String() {
			t.Errorf("line %q has no page identity: %v", line.message, line.attributes)
		}
	}
	for lineIndex, wantPostings := range map[int]int64{8: 17, 9: 11, 10: 13} {
		if gotPostings := lines.lines[lineIndex].attributes["postings"]; gotPostings != wantPostings {
			t.Errorf("line %q postings = %v, want %d",
				lines.lines[lineIndex].message, gotPostings, wantPostings)
		}
	}
}
