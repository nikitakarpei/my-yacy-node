package replydeadlines_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/replydeadlines"
)

var (
	requestArrivedAt = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	timeouts         = replydeadlines.Timeouts{
		ResponseHeader:   10 * time.Second,
		AssessmentBudget: time.Second,
	}
)

func TestReadingEndsOneAssessmentBudgetBeforeTheHeadersAreDue(t *testing.T) {
	deadlines := replydeadlines.DeadlinesFrom(requestArrivedAt, nil, timeouts)

	want := replydeadlines.Deadlines{
		ReadingEndsAt: requestArrivedAt.Add(9 * time.Second),
		HeadersDueAt:  requestArrivedAt.Add(10 * time.Second),
	}
	if deadlines != want {
		t.Fatalf("deadlines = %+v, want %+v", deadlines, want)
	}
}

func TestAShorterPreferredWaitBringsTheHeadersForward(t *testing.T) {
	deadlines := replydeadlines.DeadlinesFrom(
		requestArrivedAt, []string{"respond-async", "Wait=4, handling=lenient"}, timeouts,
	)

	if want := requestArrivedAt.Add(4 * time.Second); !deadlines.HeadersDueAt.Equal(want) {
		t.Fatalf("headers due at %v, want %v", deadlines.HeadersDueAt, want)
	}
}

func TestALongerPreferredWaitKeepsTheResponseHeaderTimeout(t *testing.T) {
	deadlines := replydeadlines.DeadlinesFrom(requestArrivedAt, []string{"wait=60"}, timeouts)

	if want := requestArrivedAt.Add(10 * time.Second); !deadlines.HeadersDueAt.Equal(want) {
		t.Fatalf("headers due at %v, want %v", deadlines.HeadersDueAt, want)
	}
}

func TestAWaitTooLargeToReadIsIgnored(t *testing.T) {
	deadlines := replydeadlines.DeadlinesFrom(
		requestArrivedAt,
		[]string{"wait=99999999999"},
		timeouts,
	)

	if want := requestArrivedAt.Add(10 * time.Second); !deadlines.HeadersDueAt.Equal(want) {
		t.Fatalf("headers due at %v, want %v", deadlines.HeadersDueAt, want)
	}
}
