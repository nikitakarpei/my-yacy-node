package yacydhtsearchpageassembly_test

import (
	"testing"

	yacydhtsearchpage "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearchpage/assembly"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := yacydhtsearchpage.LoadServiceConfig(envFrom(map[string]string{
		yacydhtsearchpage.EnvYaCyDHTSearchURL: "http://yacydhtsearch:8080",
	}))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.ListenAddr != yacydhtsearchpage.DefaultListenAddr {
		t.Fatalf("listen addr = %q", cfg.ListenAddr)
	}
	if cfg.OpsAddr != yacydhtsearchpage.DefaultOpsAddr {
		t.Fatalf("ops addr = %q", cfg.OpsAddr)
	}
	if cfg.YaCyDHTSearchURL.String() != "http://yacydhtsearch:8080" {
		t.Fatalf("search URL = %v", cfg.YaCyDHTSearchURL)
	}
}

func TestLoadServiceConfigRequiresTheYaCyDHTSearchURL(t *testing.T) {
	for _, value := range []string{"", "yacydhtsearch:8080", "ftp://yacydhtsearch/"} {
		_, err := yacydhtsearchpage.LoadServiceConfig(envFrom(map[string]string{
			yacydhtsearchpage.EnvYaCyDHTSearchURL: value,
		}))
		if err == nil {
			t.Errorf("load with search URL %q succeeded", value)
		}
	}
}
