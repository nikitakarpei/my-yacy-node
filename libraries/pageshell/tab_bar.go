package pageshell

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"slices"
	"strings"
)

var (
	errIncompleteTab        = errors.New("give each tab a title and an HTTP URL")
	errNoSingleCurrentTab   = errors.New("mark exactly one tab as current")
	errUnusableIssueTracker = errors.New("give the issue tracker an HTTP URL")
)

type TabBar struct {
	IssueTrackerURL string `json:"issueTrackerUrl"`
	Tabs            []Tab  `json:"tabs"`
}

type Tab struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Current bool   `json:"current"`
}

func TabBarFrom(setting string) (TabBar, error) {
	if strings.TrimSpace(setting) == "" {
		return TabBar{}, nil
	}
	var bar TabBar
	if err := json.Unmarshal([]byte(setting), &bar); err != nil {
		return TabBar{}, fmt.Errorf("read the tab bar: %w", err)
	}
	if err := bar.validate(); err != nil {
		return TabBar{}, err
	}
	return bar, nil
}

func (b TabBar) validate() error {
	for _, tab := range b.Tabs {
		if tab.Title == "" || !isHTTPURL(tab.URL) {
			return errIncompleteTab
		}
	}
	if len(
		slices.DeleteFunc(slices.Clone(b.Tabs), func(tab Tab) bool { return !tab.Current }),
	) != 1 {
		return errNoSingleCurrentTab
	}
	if b.IssueTrackerURL != "" && !isHTTPURL(b.IssueTrackerURL) {
		return errUnusableIssueTracker
	}
	return nil
}

func isHTTPURL(text string) bool {
	address, err := url.Parse(text)
	return err == nil && address.Host != "" &&
		(address.Scheme == "http" || address.Scheme == "https")
}

func (b TabBar) HTML() template.HTML {
	if len(b.Tabs) == 0 {
		return ""
	}
	var html strings.Builder
	html.WriteString(`<nav class="shell-tabs" aria-label="Main">`)
	for _, tab := range b.Tabs {
		html.WriteString(`<a href="` + template.HTMLEscapeString(tab.URL) + `"`)
		if tab.Current {
			html.WriteString(` aria-current="page"`)
		}
		html.WriteString(`>` + template.HTMLEscapeString(tab.Title) + `</a>`)
	}
	if b.IssueTrackerURL != "" {
		html.WriteString(
			`<a class="shell-issue-tracker" href="` + template.HTMLEscapeString(b.IssueTrackerURL) +
				`" target="_blank" rel="noopener">Issue tracker <span aria-hidden="true">↗</span></a>`,
		)
	}
	html.WriteString(`</nav>`)
	return template.HTML(html.String()) //nolint:gosec // G203: every value in the bar is escaped.
}
