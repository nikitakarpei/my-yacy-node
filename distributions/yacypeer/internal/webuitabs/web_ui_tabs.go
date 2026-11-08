// Package webuitabs gives each web UI of the peer the tab bar that links the web UIs of the peer
// in the order given, with the tab of that web UI marked.
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

type Link struct {
	Title string
	URL   string
}

type Links []Link

func (links Links) TabBarFor(current string) webuistyle.TabBar {
	tabs := make([]webuistyle.Tab, 0, len(links))
	for _, link := range links {
		tabs = append(tabs, webuistyle.Tab{
			Title: link.Title, URL: link.URL, Current: link.Title == current,
		})
	}
	return webuistyle.TabBar{IssueTrackerURL: issueTrackerURL, Tabs: tabs}
}

func (links Links) SettingFor(current string) string {
	setting, _ := json.Marshal(
		links.TabBarFor(current),
	) //nolint:errchkjson // only strings and bools.
	return string(setting)
}
