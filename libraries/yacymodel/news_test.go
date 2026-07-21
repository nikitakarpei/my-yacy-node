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

func TestNewPeerNews(t *testing.T) {
	news, err := NewPeerNews(
		WordHash("peer"),
		NewsCrawlStart,
		time.Now(),
		None[time.Time](),
		0,
		map[string]string{"url": "https://example.org"},
	)
	if err != nil {
		t.Fatalf("NewPeerNews = %v", err)
	}
	if news.Originator() != WordHash("peer") || news.Category() != NewsCrawlStart {
		t.Fatalf("NewPeerNews stored %q/%q", news.Originator(), news.Category())
	}
}

func TestNewPeerNewsRejects(t *testing.T) {
	created := time.Now()
	cases := []struct {
		originator Hash
		category   NewsCategory
	}{
		{Hash{}, NewsCrawlStart},
		{WordHash("peer"), NewsCategory{}},
	}
	for i, c := range cases {
		_, err := NewPeerNews(c.originator, c.category, created, None[time.Time](), 0, nil)
		if !errors.Is(err, ErrBadPeerNews) {
			t.Fatalf("case %d NewPeerNews = %v, want ErrBadPeerNews", i, err)
		}
	}
}
