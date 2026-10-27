package webuitabs_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/webuitabs"
)

var urls = webuitabs.URLs{
	Search:  "http://127.0.0.1:8097",
	Crawls:  "http://127.0.0.1:8093",
	Metrics: "http://127.0.0.1:9090/dashboards",
}

func TestTheTabBarLinksEveryWebUIAndMarksTheCurrentOne(t *testing.T) {
	bar := urls.TabBarFor(webuitabs.Crawls)

	want := []webuistyle.Tab{
		{Title: "Search", URL: urls.Search},
		{Title: "Crawls", URL: urls.Crawls, Current: true},
		{Title: "Metrics", URL: urls.Metrics},
	}
	if len(bar.Tabs) != len(want) {
		t.Fatalf("tabs = %+v, want %+v", bar.Tabs, want)
	}
	for position, tab := range want {
		if bar.Tabs[position] != tab {
			t.Errorf("tab %d = %+v, want %+v", position, bar.Tabs[position], tab)
		}
	}
	if bar.IssueTrackerURL == "" {
		t.Error("the tab bar links no issue tracker")
	}
}

func TestTheSettingReadsBackAsTheSameTabBar(t *testing.T) {
	bar, err := webuistyle.TabBarFrom(urls.SettingFor(webuitabs.Search))
	if err != nil {
		t.Fatalf("TabBarFrom: %v", err)
	}

	if !bar.Tabs[0].Current || bar.Tabs[2].URL != urls.Metrics {
		t.Errorf("tab bar = %+v, want the search tab current", bar)
	}
}
