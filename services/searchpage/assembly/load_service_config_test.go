package searchpageassembly_test

import (
	"strings"
	"testing"
	"time"

	searchpage "github.com/nikitakarpei/yacy-rwi-node/searchpage/assembly"
)

func everySetting() map[string]string {
	return map[string]string{
		searchpage.EnvCorpusTextURL:   "http://corpustext:8096",
		searchpage.EnvYaCySearchURL:   "http://yacydhtsearch:8080",
		searchpage.EnvVisitCrawlURL:   "http://visitcrawl:8091",
		searchpage.EnvVisitLinkSecret: "secret",
	}
}

func getenvFrom(settings map[string]string) func(string) string {
	return func(key string) string { return settings[key] }
}

func TestLoadServiceConfigReadsTheSourcesVisitCrawlAndTheDefaults(t *testing.T) {
	cfg, err := searchpage.LoadServiceConfig(getenvFrom(everySetting()))
	if err != nil {
		t.Fatalf("LoadServiceConfig: %v", err)
	}

	if cfg.CorpusTextURL.String() != "http://corpustext:8096" ||
		cfg.YaCySearchURL.String() != "http://yacydhtsearch:8080" {
		t.Errorf("config = %+v, want both sources", cfg)
	}
	if cfg.VisitCrawl == nil || cfg.VisitCrawl.URL.String() != "http://visitcrawl:8091" ||
		cfg.VisitCrawl.LinkSecret != "secret" ||
		cfg.VisitCrawl.LinkLifetime != searchpage.DefaultVisitLinkLifetime {
		t.Errorf(
			"visitcrawl config = %+v, want visitcrawl with the default lifetime",
			cfg.VisitCrawl,
		)
	}
	if cfg.ListenAddr != searchpage.DefaultListenAddr ||
		cfg.OpsAddr != searchpage.DefaultOpsAddr ||
		cfg.YaCySearchCallDeadline != searchpage.DefaultYaCySearchCallDeadline {
		t.Errorf("config = %+v, want the defaults", cfg)
	}
}

func TestLoadServiceConfigReadsTheDurationsTheOperatorSets(t *testing.T) {
	settings := everySetting()
	settings[searchpage.EnvYaCySearchCallDeadline] = "20s"
	settings[searchpage.EnvVisitLinkLifetime] = "1h"

	cfg, err := searchpage.LoadServiceConfig(getenvFrom(settings))
	if err != nil {
		t.Fatalf("LoadServiceConfig: %v", err)
	}

	if cfg.YaCySearchCallDeadline != 20*time.Second || cfg.VisitCrawl.LinkLifetime != time.Hour {
		t.Errorf("config = %+v, want the durations the operator sets", cfg)
	}
}

func TestLoadServiceConfigReadsOneSourceWithoutVisitCrawl(t *testing.T) {
	cfg, err := searchpage.LoadServiceConfig(getenvFrom(map[string]string{
		searchpage.EnvYaCySearchURL: "http://yacydhtsearch:8080",
	}))
	if err != nil {
		t.Fatalf("LoadServiceConfig: %v", err)
	}

	if cfg.CorpusTextURL != nil || cfg.YaCySearchURL == nil || cfg.VisitCrawl != nil {
		t.Errorf("config = %+v, want only the YaCy network", cfg)
	}
}

func TestLoadServiceConfigReadsTheTabBar(t *testing.T) {
	settings := everySetting()
	settings[searchpage.EnvTabBar] = `{"tabs": [{"title": "Search", "url": "http://search:8097", "current": true}]}`

	cfg, err := searchpage.LoadServiceConfig(getenvFrom(settings))
	if err != nil {
		t.Fatalf("LoadServiceConfig: %v", err)
	}

	if len(cfg.TabBar.Tabs) != 1 || cfg.TabBar.Tabs[0].URL != "http://search:8097" {
		t.Errorf("tab bar = %+v, want the search tab", cfg.TabBar)
	}
}

func TestLoadServiceConfigRefusesATabBarItCannotShow(t *testing.T) {
	settings := everySetting()
	settings[searchpage.EnvTabBar] = `{"tabs": [{"title": "Search", "url": "http://search:8097"}]}`

	_, err := searchpage.LoadServiceConfig(getenvFrom(settings))
	if err == nil || !strings.Contains(err.Error(), searchpage.EnvTabBar) {
		t.Errorf("LoadServiceConfig = %v, want %s named", err, searchpage.EnvTabBar)
	}
}

func TestLoadServiceConfigRefusesNoSource(t *testing.T) {
	settings := everySetting()
	delete(settings, searchpage.EnvCorpusTextURL)
	delete(settings, searchpage.EnvYaCySearchURL)

	_, err := searchpage.LoadServiceConfig(getenvFrom(settings))
	if err == nil || !strings.Contains(err.Error(), searchpage.EnvCorpusTextURL) ||
		!strings.Contains(err.Error(), searchpage.EnvYaCySearchURL) {
		t.Errorf("LoadServiceConfig = %v, want both sources named", err)
	}
}

func TestLoadServiceConfigRefusesHalfOfVisitCrawl(t *testing.T) {
	for _, name := range []string{searchpage.EnvVisitCrawlURL, searchpage.EnvVisitLinkSecret} {
		t.Run(name, func(t *testing.T) {
			settings := everySetting()
			delete(settings, name)

			_, err := searchpage.LoadServiceConfig(getenvFrom(settings))
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Errorf("LoadServiceConfig = %v, want %s named", err, name)
			}
		})
	}
}

func TestLoadServiceConfigRefusesAURLThatIsNotAWebAddress(t *testing.T) {
	for _, name := range []string{
		searchpage.EnvCorpusTextURL, searchpage.EnvYaCySearchURL, searchpage.EnvVisitCrawlURL,
	} {
		t.Run(name, func(t *testing.T) {
			settings := everySetting()
			settings[name] = "ftp://elsewhere"

			_, err := searchpage.LoadServiceConfig(getenvFrom(settings))
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Errorf("LoadServiceConfig = %v, want %s named", err, name)
			}
		})
	}
}

func TestLoadServiceConfigRefusesADurationThatIsNotPositive(t *testing.T) {
	for _, name := range []string{
		searchpage.EnvYaCySearchCallDeadline, searchpage.EnvVisitLinkLifetime,
	} {
		t.Run(name, func(t *testing.T) {
			settings := everySetting()
			settings[name] = "0s"

			_, err := searchpage.LoadServiceConfig(getenvFrom(settings))
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Errorf("LoadServiceConfig = %v, want %s named", err, name)
			}
		})
	}
}
