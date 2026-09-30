package responsedeadline_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/responsedeadline"
)

const responseHeaderTimeout = 10 * time.Second

var requestArrivedAt = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

func TestTheHeadersAreDueOneResponseHeaderTimeoutAfterTheRequest(t *testing.T) {
	headersDeadline := responsedeadline.HeadersDeadlineFrom(
		requestArrivedAt,
		nil,
		responseHeaderTimeout,
	)

	if want := requestArrivedAt.Add(10 * time.Second); !headersDeadline.Equal(want) {
		t.Fatalf("headers deadline %v, want %v", headersDeadline, want)
	}
}

func TestAShorterPreferredWaitBringsTheHeadersForward(t *testing.T) {
	headersDeadline := responsedeadline.HeadersDeadlineFrom(
		requestArrivedAt,
		[]string{"respond-async", "Wait=4, handling=lenient"},
		responseHeaderTimeout,
	)

	if want := requestArrivedAt.Add(4 * time.Second); !headersDeadline.Equal(want) {
		t.Fatalf("headers deadline %v, want %v", headersDeadline, want)
	}
}

func TestALongerPreferredWaitKeepsTheResponseHeaderTimeout(t *testing.T) {
	headersDeadline := responsedeadline.HeadersDeadlineFrom(
		requestArrivedAt,
		[]string{"wait=60"},
		responseHeaderTimeout,
	)

	if want := requestArrivedAt.Add(10 * time.Second); !headersDeadline.Equal(want) {
		t.Fatalf("headers deadline %v, want %v", headersDeadline, want)
	}
}

func TestAWaitTooLargeToReadIsIgnored(t *testing.T) {
	headersDeadline := responsedeadline.HeadersDeadlineFrom(
		requestArrivedAt,
		[]string{"wait=99999999999"},
		responseHeaderTimeout,
	)

	if want := requestArrivedAt.Add(10 * time.Second); !headersDeadline.Equal(want) {
		t.Fatalf("headers deadline %v, want %v", headersDeadline, want)
	}
}
