package replydeadline_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/replydeadline"
)

const responseHeaderTimeout = 10 * time.Second

var requestArrivedAt = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

func TestTheHeadersAreDueOneResponseHeaderTimeoutAfterTheRequest(t *testing.T) {
	headersDueAt := replydeadline.HeadersDueAtFrom(requestArrivedAt, nil, responseHeaderTimeout)

	if want := requestArrivedAt.Add(10 * time.Second); !headersDueAt.Equal(want) {
		t.Fatalf("headers due at %v, want %v", headersDueAt, want)
	}
}

func TestAShorterPreferredWaitBringsTheHeadersForward(t *testing.T) {
	headersDueAt := replydeadline.HeadersDueAtFrom(
		requestArrivedAt,
		[]string{"respond-async", "Wait=4, handling=lenient"},
		responseHeaderTimeout,
	)

	if want := requestArrivedAt.Add(4 * time.Second); !headersDueAt.Equal(want) {
		t.Fatalf("headers due at %v, want %v", headersDueAt, want)
	}
}

func TestALongerPreferredWaitKeepsTheResponseHeaderTimeout(t *testing.T) {
	headersDueAt := replydeadline.HeadersDueAtFrom(
		requestArrivedAt,
		[]string{"wait=60"},
		responseHeaderTimeout,
	)

	if want := requestArrivedAt.Add(10 * time.Second); !headersDueAt.Equal(want) {
		t.Fatalf("headers due at %v, want %v", headersDueAt, want)
	}
}

func TestAWaitTooLargeToReadIsIgnored(t *testing.T) {
	headersDueAt := replydeadline.HeadersDueAtFrom(
		requestArrivedAt,
		[]string{"wait=99999999999"},
		responseHeaderTimeout,
	)

	if want := requestArrivedAt.Add(10 * time.Second); !headersDueAt.Equal(want) {
		t.Fatalf("headers due at %v, want %v", headersDueAt, want)
	}
}
