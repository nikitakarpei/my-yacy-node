package yacymodel

import (
	"errors"
	"testing"
	"time"
)

func TestParseNewsCategory(t *testing.T) {
	got, err := ParseNewsCategory("crwlstrt")
	if err != nil || got != NewsCrawlStart {
		t.Fatalf("ParseNewsCategory = %q, %v", got, err)
	}
}

func TestParseNewsCategoryRejects(t *testing.T) {
	if _, err := ParseNewsCategory("nope"); !errors.Is(err, ErrBadNewsCategory) {
		t.Fatalf("ParseNewsCategory = %v, want ErrBadNewsCategory", err)
	}
}

func TestPeerNewsValidate(t *testing.T) {
	news := PeerNews{
		Originator: WordHash("peer"),
		Category:   NewsCrawlStart,
		Created:    time.Now(),
		Attributes: map[string]string{"url": "https://example.org"},
	}
	if err := news.Validate(); err != nil {
		t.Fatalf("Validate = %v", err)
	}
}

func TestPeerNewsValidateRejects(t *testing.T) {
	bad := []PeerNews{
		{Originator: Hash("short"), Category: NewsCrawlStart},
		{Originator: WordHash("peer"), Category: "nope"},
		{
			Originator: WordHash("peer"),
			Category:   NewsCrawlStart,
			Attributes: map[string]string{"k": string(make([]byte, 1000))},
		},
	}
	for i, news := range bad {
		if err := news.Validate(); !errors.Is(err, ErrBadPeerNews) {
			t.Fatalf("case %d Validate = %v, want ErrBadPeerNews", i, err)
		}
	}
}
