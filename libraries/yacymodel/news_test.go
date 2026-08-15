package yacymodel_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestParseNewsCategory(t *testing.T) {
	got, err := yacymodel.ParseNewsCategory("crwlstrt")
	if err != nil || got != yacymodel.NewsCrawlStart {
		t.Fatalf("ParseNewsCategory = %q, %v", got, err)
	}
}

func TestParseNewsCategoryRejects(t *testing.T) {
	if _, err := yacymodel.ParseNewsCategory(
		"nope",
	); !errors.Is(
		err,
		yacymodel.ErrBadNewsCategory,
	) {
		t.Fatalf("ParseNewsCategory = %v, want ErrBadNewsCategory", err)
	}
}

func TestNewPeerNews(t *testing.T) {
	news, err := yacymodel.NewPeerNews(
		yacymodel.WordHash("peer"),
		yacymodel.NewsCrawlStart,
		time.Now(),
		yacymodel.None[time.Time](),
		0,
		map[string]string{"url": "https://example.org"},
	)
	if err != nil {
		t.Fatalf("NewPeerNews = %v", err)
	}
	if news.Originator() != yacymodel.WordHash("peer") ||
		news.Category() != yacymodel.NewsCrawlStart {
		t.Fatalf("NewPeerNews stored %q/%q", news.Originator(), news.Category())
	}
}

func TestNewPeerNewsRejects(t *testing.T) {
	created := time.Now()
	cases := []struct {
		originator yacymodel.Hash
		category   yacymodel.NewsCategory
	}{
		{yacymodel.Hash{}, yacymodel.NewsCrawlStart},
		{yacymodel.WordHash("peer"), yacymodel.NewsCategory{}},
	}
	for i, c := range cases {
		_, err := yacymodel.NewPeerNews(
			c.originator,
			c.category,
			created,
			yacymodel.None[time.Time](),
			0,
			nil,
		)
		if !errors.Is(err, yacymodel.ErrBadPeerNews) {
			t.Fatalf("case %d NewPeerNews = %v, want ErrBadPeerNews", i, err)
		}
	}
}
