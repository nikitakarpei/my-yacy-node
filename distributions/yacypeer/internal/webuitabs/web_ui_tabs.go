// Package webuitabs gives each web UI of the peer the tab bar that links the search page, the
// crawl console, the Companions pages when the peer has them, and the dashboards, with the tab
// of that web UI marked.
package webuitabs

import (
	"encoding/json"

	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
)

const issueTrackerURL = "https://github.com/nikitakarpei/my-yacy-node/issues"

const (
	Search     = "Search"
	Crawls     = "Crawls"
	Companions = "Companions"
	Metrics    = "Metrics"
)

type URLs struct {
	Search     string
	Crawls     string
	Companions string
	Metrics    string
}

func (urls URLs) TabBarFor(current string) webuistyle.TabBar {
	tabs := []webuistyle.Tab{
		{Title: Search, URL: urls.Search, Current: current == Search},
		{Title: Crawls, URL: urls.Crawls, Current: current == Crawls},
	}
	if urls.Companions != "" {
		tabs = append(tabs, webuistyle.Tab{
			Title: Companions, URL: urls.Companions, Current: current == Companions,
		})
	}
	tabs = append(
		tabs,
		webuistyle.Tab{Title: Metrics, URL: urls.Metrics, Current: current == Metrics},
	)
	return webuistyle.TabBar{IssueTrackerURL: issueTrackerURL, Tabs: tabs}
}

func (urls URLs) SettingFor(current string) string {
	setting, _ := json.Marshal(
		urls.TabBarFor(current),
	) //nolint:errchkjson // only strings and bools.
	return string(setting)
}
