package text_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/cdxscrape/internal/scraperequests/text"
)

func TestPublishWritesOneScrapeRequestPerLine(t *testing.T) {
	requests := &bytes.Buffer{}
	publisher := text.New(requests)

	for _, requested := range []string{"http://pywb:8080/a", "http://pywb:8080/b"} {
		if err := publisher.Publish(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, requested),
		); err != nil {
			t.Fatalf("publish %s: %v", requested, err)
		}
	}
	publisher.Close()

	wanted := "http://pywb:8080/a\nhttp://pywb:8080/b\n"
	if requests.String() != wanted {
		t.Fatalf("requests = %q, want %q", requests.String(), wanted)
	}
}

func TestPublishFailsWhenTheRequestsCannotBeWritten(t *testing.T) {
	publisher := text.New(refusedWrites{})

	err := publisher.Publish(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://pywb:8080/a"),
	)
	if err == nil {
		t.Fatal("publish: want an error")
	}
}

type refusedWrites struct{}

func (refusedWrites) Write([]byte) (int, error) { return 0, errors.New("refused") }
