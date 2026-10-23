package main_test

import (
	"testing"

	crawlconsole "github.com/nikitakarpei/yacy-rwi-node/crawlconsole/cmd/crawlconsole"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := crawlconsole.LoadServiceConfig(envFrom(map[string]string{
		crawlconsole.EnvCrawlerAPIURL: "http://yacycrawler:9095",
	}))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.ListenAddr != crawlconsole.DefaultListenAddr {
		t.Fatalf("listen addr = %q", cfg.ListenAddr)
	}
	if cfg.OpsAddr != crawlconsole.DefaultOpsAddr {
		t.Fatalf("ops addr = %q", cfg.OpsAddr)
	}
	if cfg.CrawlerAPIURL.String() != "http://yacycrawler:9095" {
		t.Fatalf("crawler API URL = %v", cfg.CrawlerAPIURL)
	}
}

func TestLoadServiceConfigRequiresTheCrawlerAPIURL(t *testing.T) {
	for _, value := range []string{"", "yacycrawler:9095", "ftp://yacycrawler/"} {
		_, err := crawlconsole.LoadServiceConfig(envFrom(map[string]string{
			crawlconsole.EnvCrawlerAPIURL: value,
		}))
		if err == nil {
			t.Errorf("load with crawler API URL %q succeeded", value)
		}
	}
}
