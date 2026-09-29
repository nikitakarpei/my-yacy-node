package main_test

import (
	"net/url"
	"reflect"
	"testing"
	"time"

	spamproxy "github.com/nikitakarpei/yacy-rwi-node/spamproxy/cmd/spamproxy"
)

var requiredEnvironment = map[string]string{
	spamproxy.EnvEgressProxyURL:       "http://squid:3128",
	spamproxy.EnvSafetensorsModelPath: "/model/spam-model.safetensors",
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
	cfg, err := spamproxy.LoadServiceConfig(getenvFrom(requiredEnvironment))
	if err != nil {
		t.Fatal(err)
	}

	want := spamproxy.ServiceConfig{
		ListenAddr:            ":8080",
		EgressProxyURL:        &url.URL{Scheme: "http", Host: "squid:3128"},
		EgressProxyDialMode:   spamproxy.DialModeTunnel,
		SafetensorsModelPath:  "/model/spam-model.safetensors",
		PageByteCeiling:       1048576,
		MaxPagesReadAtOnce:    64,
		ResponseHeaderTimeout: 10 * time.Second,
		RelayIdleTimeout:      30 * time.Second,
		OpsAddr:               ":9090",
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Fatalf("config = %+v, want %+v", cfg, want)
	}
}

func TestSetSettingsReplaceTheirDefaults(t *testing.T) {
	cfg, err := spamproxy.LoadServiceConfig(getenvFrom(map[string]string{
		spamproxy.EnvEgressProxyDialMode:   "absolute-url",
		spamproxy.EnvMaxPagesReadAtOnce:    "2",
		spamproxy.EnvResponseHeaderTimeout: "1m",
		spamproxy.EnvRelayIdleTimeout:      "1500ms",
	}, requiredEnvironment))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.EgressProxyDialMode != spamproxy.DialModeAbsoluteURL ||
		cfg.MaxPagesReadAtOnce != 2 ||
		cfg.ResponseHeaderTimeout != time.Minute ||
		cfg.RelayIdleTimeout != 1500*time.Millisecond {
		t.Fatalf("config = %+v", cfg)
	}
}

func TestAMissingRequiredSettingIsRefused(t *testing.T) {
	for name := range requiredEnvironment {
		if _, err := spamproxy.LoadServiceConfig(
			getenvFrom(map[string]string{name: ""}, requiredEnvironment),
		); err == nil {
			t.Errorf("%s: config loaded without it", name)
		}
	}
}

func TestAMalformedSettingIsRefused(t *testing.T) {
	malformedSettings := map[string]string{
		spamproxy.EnvPageByteCeiling:       "0",
		spamproxy.EnvMaxPagesReadAtOnce:    "many",
		spamproxy.EnvRelayIdleTimeout:      "30",
		spamproxy.EnvResponseHeaderTimeout: "soon",
		spamproxy.EnvEgressProxyDialMode:   "socks",
	}
	for name, value := range malformedSettings {
		if _, err := spamproxy.LoadServiceConfig(
			getenvFrom(map[string]string{name: value}, requiredEnvironment),
		); err == nil {
			t.Errorf("%s=%s: config loaded", name, value)
		}
	}
}
