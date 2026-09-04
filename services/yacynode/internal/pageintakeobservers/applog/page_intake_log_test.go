package applog_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	pageintakeobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageintakeobservers/applog"
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

func TestPageIntakeLogWritesEveryIntakeFactAtItsOperationalLevel(t *testing.T) {
	lines := &recordedLogLines{Handler: slog.Default().Handler()}
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(lines))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	ctx := context.Background()
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.test/page")
	cause := errors.New("intake failed")
	observer := pageintakeobserversapplog.PageIntakeLog{}

	observer.OfferedPageInvalid(ctx)
	observer.PageOffered(ctx, "message", pageURL)
	observer.DocumentExtractionFailed(ctx, "message", pageURL, cause)
	observer.NoIndexDerived(ctx, "message", pageURL)
	observer.URLMetadataAdmitted(ctx, "message", pageURL)
	observer.URLMetadataAdmissionBusy(ctx, "message", pageURL)
	observer.URLMetadataAdmissionFailed(ctx, "message", pageURL, cause)
	observer.PostingsAdmitted(ctx, "message", pageURL, 17)
	observer.PostingsAdmissionBusy(ctx, "message", pageURL, 11)
	observer.PostingsAdmissionFailed(ctx, "message", pageURL, 13, cause)
	observer.PageIndexed(ctx, "message", pageURL)

	firstLine := lines.lines[0]
	if firstLine.level != slog.LevelWarn {
		t.Errorf("line %q level = %s, want %s", firstLine.message, firstLine.level, slog.LevelWarn)
	}

	remainingLines := lines.lines[1:]
	wantLevels := []slog.Level{
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
	if len(remainingLines) != len(wantLevels) {
		t.Fatalf("logged %d lines, want %d", len(remainingLines), len(wantLevels))
	}
	for lineIndex, wantLevel := range wantLevels {
		line := remainingLines[lineIndex]
		if line.level != wantLevel {
			t.Errorf("line %q level = %s, want %s", line.message, line.level, wantLevel)
		}
		if line.attributes["message"] != "message" {
			t.Errorf("line %q message identity = %v, want message", line.message,
				line.attributes["message"])
		}
		if line.attributes["pageUrl"] != pageURL.String() {
			t.Errorf("line %q has no page identity: %v", line.message, line.attributes)
		}
	}
	for lineIndex, wantPostings := range map[int]int64{6: 17, 7: 11, 8: 13} {
		if gotPostings := remainingLines[lineIndex].attributes["postings"]; gotPostings != wantPostings {
			t.Errorf("line %q postings = %v, want %d",
				remainingLines[lineIndex].message, gotPostings, wantPostings)
		}
	}
}
