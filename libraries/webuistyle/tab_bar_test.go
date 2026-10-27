package webuistyle_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
)

const everyTab = `{
	"tabs": [
		{"title": "Search", "url": "http://127.0.0.1:8097", "current": true},
		{"title": "Crawls", "url": "https://console.example/crawls"}
	],
	"issueTrackerUrl": "https://code.example/issues"
}`

func TestTabBarFromShowsEachTabAndMarksTheCurrentOne(t *testing.T) {
	bar, err := webuistyle.TabBarFrom(everyTab)
	if err != nil {
		t.Fatalf("TabBarFrom: %v", err)
	}

	html := string(bar.HTML())

	for _, want := range []string{
		`<a href="http://127.0.0.1:8097" aria-current="page">Search</a>`,
		`<a href="https://console.example/crawls">Crawls</a>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("tab bar = %s, want %s", html, want)
		}
	}
}

func TestTabBarFromOpensTheIssueTrackerInANewTab(t *testing.T) {
	bar, err := webuistyle.TabBarFrom(everyTab)
	if err != nil {
		t.Fatalf("TabBarFrom: %v", err)
	}

	html := string(bar.HTML())

	if !strings.Contains(
		html,
		`href="https://code.example/issues" target="_blank" rel="noopener">Issue tracker`,
	) {
		t.Errorf("tab bar = %s, want the issue tracker in a new tab", html)
	}
}

func TestTabBarFromWithoutASettingShowsNoBar(t *testing.T) {
	bar, err := webuistyle.TabBarFrom(" ")
	if err != nil {
		t.Fatalf("TabBarFrom: %v", err)
	}

	if html := bar.HTML(); html != "" {
		t.Errorf("tab bar = %s, want none", html)
	}
}

func TestTabBarFromRefusesABarItCannotShow(t *testing.T) {
	for name, setting := range map[string]string{
		"not JSON":         `Search`,
		"no title":         `{"tabs": [{"url": "http://search", "current": true}]}`,
		"no web address":   `{"tabs": [{"title": "Search", "url": "ftp://search", "current": true}]}`,
		"no host":          `{"tabs": [{"title": "Search", "url": "/search", "current": true}]}`,
		"no address":       `{"tabs": [{"title": "Search", "url": "http://%zz", "current": true}]}`,
		"no current tab":   `{"tabs": [{"title": "Search", "url": "http://search"}]}`,
		"two current tabs": `{"tabs": [{"title": "A", "url": "http://a", "current": true}, {"title": "B", "url": "http://b", "current": true}]}`,
		"no issue tracker": `{"tabs": [{"title": "Search", "url": "http://search", "current": true}], "issueTrackerUrl": "issues"}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := webuistyle.TabBarFrom(setting); err == nil {
				t.Errorf("TabBarFrom(%s) = nil, want an error", setting)
			}
		})
	}
}

func TestTabBarEscapesItsTitles(t *testing.T) {
	bar := webuistyle.TabBar{Tabs: []webuistyle.Tab{
		{Title: "<b>Search</b>", URL: "http://search", Current: true},
	}}

	if html := string(bar.HTML()); strings.Contains(html, "<b>") {
		t.Errorf("tab bar = %s, want the title escaped", html)
	}
}
