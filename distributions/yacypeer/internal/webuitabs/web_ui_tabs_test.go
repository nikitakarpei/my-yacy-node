package webuitabs_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/webuitabs"
)

var links = webuitabs.Links{
	{Title: webuitabs.Search, URL: "http://127.0.0.1:8097"},
	{Title: webuitabs.Crawls, URL: "http://127.0.0.1:8093"},
	{Title: webuitabs.Metrics, URL: "http://127.0.0.1:9090/dashboards"},
}

func TestTheTabBarLinksEveryWebUIInOrderAndMarksTheCurrentOne(t *testing.T) {
	bar := links.TabBarFor(webuitabs.Crawls)

	want := []webuistyle.Tab{
		{Title: "Search", URL: links[0].URL},
		{Title: "Crawls", URL: links[1].URL, Current: true},
		{Title: "Metrics", URL: links[2].URL},
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
	bar, err := webuistyle.TabBarFrom(links.SettingFor(webuitabs.Search))
	if err != nil {
		t.Fatalf("TabBarFrom: %v", err)
	}

	if !bar.Tabs[0].Current || bar.Tabs[2].URL != links[2].URL {
		t.Errorf("tab bar = %+v, want the search tab current", bar)
	}
}
