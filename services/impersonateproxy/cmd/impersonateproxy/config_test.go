package main_test

import (
	"net/url"
	"reflect"
	"testing"
	"time"

	impersonateproxy "github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/cmd/impersonateproxy"
)

var requiredEnvironment = map[string]string{
	impersonateproxy.EnvEgressProxyURL: "http://smokescreen:4750",
}

func getenvFrom(environments ...map[string]string) func(string) string {
	return func(name string) string {
		for _, environment := range environments {
			if value, found := environment[name]; found {
				return value
			}
		}
		return ""
	}
}

func TestUnsetSettingsTakeTheirDefaults(t *testing.T) {
	cfg, err := impersonateproxy.LoadServiceConfig(getenvFrom(requiredEnvironment))
	if err != nil {
		t.Fatal(err)
	}

	want := impersonateproxy.ServiceConfig{
		ListenAddr:     ":8080",
		EgressProxyURL: &url.URL{Scheme: "http", Host: "smokescreen:4750"},
		FetchTimeout:   30 * time.Second,
		DefaultReferer: "",
		OpsAddr:        ":9090",
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Fatalf("config = %+v, want %+v", cfg, want)
	}
}

func TestSetSettingsReplaceTheirDefaults(t *testing.T) {
	cfg, err := impersonateproxy.LoadServiceConfig(getenvFrom(map[string]string{
		impersonateproxy.EnvListenAddr:     ":8181",
		impersonateproxy.EnvFetchTimeout:   "1m",
		impersonateproxy.EnvDefaultReferer: "https://www.google.com/",
		impersonateproxy.EnvOpsAddr:        ":9191",
	}, requiredEnvironment))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.ListenAddr != ":8181" || cfg.FetchTimeout != time.Minute ||
		cfg.DefaultReferer != "https://www.google.com/" || cfg.OpsAddr != ":9191" {
		t.Fatalf("config = %+v", cfg)
	}
}

func TestAMissingEgressProxyIsRefused(t *testing.T) {
	if _, err := impersonateproxy.LoadServiceConfig(getenvFrom(map[string]string{})); err == nil {
		t.Fatal("config loaded without an egress proxy")
	}
}

func TestAMalformedSettingIsRefused(t *testing.T) {
	malformedSettings := map[string]string{
		impersonateproxy.EnvFetchTimeout:   "30",
		impersonateproxy.EnvDefaultReferer: "www.google.com",
		impersonateproxy.EnvEgressProxyURL: "smokescreen:4750",
	}
	for name, value := range malformedSettings {
		if _, err := impersonateproxy.LoadServiceConfig(
			getenvFrom(map[string]string{name: value}, requiredEnvironment),
		); err == nil {
			t.Errorf("%s=%s: config loaded", name, value)
		}
	}
}
