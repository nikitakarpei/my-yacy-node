package pagemarkdownstore_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
)

func TestScrapeOutcomeNoticeRoundTrip(t *testing.T) {
	for _, outcome := range []pagemarkdownstore.ScrapeOutcome{
		pagemarkdownstore.MarkdownStored,
		pagemarkdownstore.PageGivenUp,
	} {
		t.Run(string(outcome), func(t *testing.T) {
			notice := pagemarkdownstore.ScrapeOutcomeNotice{
				RequestedURL: canonicalurltest.CanonicalURLOf(t, "https://example.test/page"),
				Outcome:      outcome,
			}

			data, err := pagemarkdownstore.MarshalScrapeOutcomeNotice(notice)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			got, err := pagemarkdownstore.UnmarshalScrapeOutcomeNotice(data)
			if err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got != notice {
				t.Errorf("round-trip mismatch:\nwant %#v\ngot  %#v", notice, got)
			}
		})
	}
}

func TestUnmarshalScrapeOutcomeNoticeRejectsAnUnknownOutcome(t *testing.T) {
	data := []byte(`{"RequestedURL":"https://example.test/page","Outcome":"scraped-somehow"}`)

	if _, err := pagemarkdownstore.UnmarshalScrapeOutcomeNotice(data); err == nil {
		t.Fatal("want an error for an unknown outcome")
	}
}

func TestUnmarshalScrapeOutcomeNoticeInvalidJSON(t *testing.T) {
	if _, err := pagemarkdownstore.UnmarshalScrapeOutcomeNotice([]byte("not json")); err == nil {
		t.Fatal("want an error for invalid JSON")
	}
}

func TestScrapeOutcomeSubjectOfDistinguishesURLs(t *testing.T) {
	if pagemarkdownstore.ScrapeOutcomeSubjectOf(
		canonicalurltest.CanonicalURLOf(t, "https://example.test/a"),
	) ==
		pagemarkdownstore.ScrapeOutcomeSubjectOf(
			canonicalurltest.CanonicalURLOf(t, "https://example.test/b"),
		) {
		t.Fatal("distinct URLs collided to one subject")
	}
}

func TestScrapeOutcomeSubjectOfIsUnderTheSubjectEveryNoticeIsSentOn(t *testing.T) {
	subject := pagemarkdownstore.ScrapeOutcomeSubjectOf(
		canonicalurltest.CanonicalURLOf(t, "https://example.test/page"),
	)

	if !strings.HasPrefix(subject, pagemarkdownstore.ScrapeOutcomeSubjectPrefix+".") {
		t.Fatalf(
			"subject %q is not under %q",
			subject,
			pagemarkdownstore.ScrapeOutcomeSubjectPrefix,
		)
	}
}
