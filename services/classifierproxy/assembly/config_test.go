package classifierproxyassembly_test

import (
	"net/url"
	"reflect"
	"testing"
	"time"

	classifierproxy "github.com/nikitakarpei/yacy-rwi-node/classifierproxy/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
)

var requiredEnvironment = map[string]string{
	classifierproxy.EnvEgressProxyURL:        "http://squid:3128",
	classifierproxy.EnvSafetensorsModelPaths: "spam=/model/spam.safetensors",
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
	cfg, err := classifierproxy.LoadServiceConfig(getenvFrom(requiredEnvironment))
	if err != nil {
		t.Fatal(err)
	}

	want := classifierproxy.ServiceConfig{
		ListenAddr:          ":8080",
		EgressProxyURL:      &url.URL{Scheme: "http", Host: "squid:3128"},
		EgressProxyDialMode: classifierproxy.DialModeTunnel,
		SafetensorsModelPathPerCategory: map[pageclassification.Category]string{
			pageclassification.Spam: "/model/spam.safetensors",
		},
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
	cfg, err := classifierproxy.LoadServiceConfig(getenvFrom(map[string]string{
		classifierproxy.EnvEgressProxyDialMode:   "absolute-url",
		classifierproxy.EnvMaxPagesReadAtOnce:    "2",
		classifierproxy.EnvResponseHeaderTimeout: "1m",
		classifierproxy.EnvRelayIdleTimeout:      "1500ms",
	}, requiredEnvironment))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.EgressProxyDialMode != classifierproxy.DialModeAbsoluteURL ||
		cfg.MaxPagesReadAtOnce != 2 ||
		cfg.ResponseHeaderTimeout != time.Minute ||
		cfg.RelayIdleTimeout != 1500*time.Millisecond {
		t.Fatalf("config = %+v", cfg)
	}
}

func TestEachCategoryNamesTheFileOfItsModel(t *testing.T) {
	cfg, err := classifierproxy.LoadServiceConfig(getenvFrom(map[string]string{
		classifierproxy.EnvSafetensorsModelPaths: " spam=/models/spam.safetensors , adult = /models/adult.safetensors",
	}, requiredEnvironment))
	if err != nil {
		t.Fatal(err)
	}

	if want := map[pageclassification.Category]string{
		pageclassification.Spam: "/models/spam.safetensors",
		"adult":                 "/models/adult.safetensors",
	}; !reflect.DeepEqual(cfg.SafetensorsModelPathPerCategory, want) {
		t.Fatalf("model paths = %v, want %v", cfg.SafetensorsModelPathPerCategory, want)
	}
}

func TestAMissingRequiredSettingIsRefused(t *testing.T) {
	for name := range requiredEnvironment {
		if _, err := classifierproxy.LoadServiceConfig(
			getenvFrom(map[string]string{name: ""}, requiredEnvironment),
		); err == nil {
			t.Errorf("%s: config loaded without it", name)
		}
	}
}

func TestAMalformedSettingIsRefused(t *testing.T) {
	malformedSettings := map[string]string{
		classifierproxy.EnvPageByteCeiling:       "0",
		classifierproxy.EnvMaxPagesReadAtOnce:    "many",
		classifierproxy.EnvRelayIdleTimeout:      "30",
		classifierproxy.EnvResponseHeaderTimeout: "soon",
		classifierproxy.EnvEgressProxyDialMode:   "socks",
	}
	for name, value := range malformedSettings {
		if _, err := classifierproxy.LoadServiceConfig(
			getenvFrom(map[string]string{name: value}, requiredEnvironment),
		); err == nil {
			t.Errorf("%s=%s: config loaded", name, value)
		}
	}
}

func TestAMalformedModelPathOfACategoryIsRefused(t *testing.T) {
	for _, value := range []string{
		"/model/spam.safetensors",
		"spam=",
		"=/model/spam.safetensors",
		"Spam=/model/spam.safetensors",
		"spam=/model/a.safetensors,spam=/model/b.safetensors",
	} {
		if _, err := classifierproxy.LoadServiceConfig(getenvFrom(map[string]string{
			classifierproxy.EnvSafetensorsModelPaths: value,
		}, requiredEnvironment)); err == nil {
			t.Errorf("%s: config loaded", value)
		}
	}
}
