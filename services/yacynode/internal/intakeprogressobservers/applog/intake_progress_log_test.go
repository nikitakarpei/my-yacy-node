package applog_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	intakeprogressobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/intakeprogressobservers/applog"
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

func TestIntakeProgressLogWritesEveryIntakeFactAtItsOperationalLevel(t *testing.T) {
	lines := &recordedLogLines{Handler: slog.Default().Handler()}
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(lines))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	ctx := context.Background()
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.test/page")
	cause := errors.New("intake failed")
	progress := intakeprogressobserversapplog.IntakeProgressLog{}

	progress.OfferedPageInvalid(ctx)
	progress.PageOffered(ctx, "message", pageURL)
	progress.DocumentExtractionFailed(ctx, "message", pageURL, cause)
	progress.NoIndexDerived(ctx, "message", pageURL)
	progress.URLMetadataAdmitted(ctx, "message", pageURL)
	progress.URLMetadataAdmissionBusy(ctx, "message", pageURL)
	progress.URLMetadataAdmissionFailed(ctx, "message", pageURL, cause)
	progress.PostingsAdmitted(ctx, "message", pageURL, 17)
	progress.PostingsAdmissionBusy(ctx, "message", pageURL, 11)
	progress.PostingsAdmissionFailed(ctx, "message", pageURL, 13, cause)
	progress.PageIndexed(ctx, "message", pageURL)

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
		if line.attributes["pageUrl"] != pageURL.String() {
			t.Errorf("line %q has no page identity: %v", line.message, line.attributes)
		}
	}
	for lineIndex, wantPostings := range map[int]int64{6: 17, 7: 11, 8: 13} {
		if gotPostings := lines.lines[lineIndex].attributes["postings"]; gotPostings != wantPostings {
			t.Errorf("line %q postings = %v, want %d",
				lines.lines[lineIndex].message, gotPostings, wantPostings)
		}
	}
}
