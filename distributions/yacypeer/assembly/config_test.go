package yacypeerassembly_test

import (
	"testing"

	yacypeerassembly "github.com/nikitakarpei/yacy-rwi-node/yacypeer/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadServiceConfigRequiresTheSeedlists(t *testing.T) {
	if _, err := yacypeerassembly.LoadServiceConfig(envFrom(nil)); err == nil {
		t.Fatal("config loaded without seedlists, want an error")
	}
}

func TestLoadServiceConfigFallsBackToTheDefaultOfEverySetting(t *testing.T) {
	cfg, err := yacypeerassembly.LoadServiceConfig(envFrom(map[string]string{
		yacypeerassembly.EnvSeedlistURLs: "https://seeds.example/seed.txt",
	}))
	if err != nil {
		t.Fatalf("load the config: %v", err)
	}
	want := yacypeerassembly.ServiceConfig{
		DataDirectory: yacypeerassembly.DefaultDataDirectory,
		NetworkName:   yacyproto.DefaultNetwork,
		SeedlistURLs:  "https://seeds.example/seed.txt",
		OpsAddr:       yacypeerassembly.DefaultOpsAddr,
	}
	if cfg != want {
		t.Errorf("config = %+v, want %+v", cfg, want)
	}
}

func TestLoadServiceConfigTakesEveryNamedSetting(t *testing.T) {
	cfg, err := yacypeerassembly.LoadServiceConfig(envFrom(map[string]string{
		yacypeerassembly.EnvSeedlistURLs:  "https://seeds.example/seed.txt",
		yacypeerassembly.EnvDataDirectory: "/var/lib/yacypeer",
		yacypeerassembly.EnvNetworkName:   "realm",
		yacypeerassembly.EnvOpsAddr:       "127.0.0.1:9099",
	}))
	if err != nil {
		t.Fatalf("load the config: %v", err)
	}
	want := yacypeerassembly.ServiceConfig{
		DataDirectory: "/var/lib/yacypeer",
		NetworkName:   "realm",
		SeedlistURLs:  "https://seeds.example/seed.txt",
		OpsAddr:       "127.0.0.1:9099",
	}
	if cfg != want {
		t.Errorf("config = %+v, want %+v", cfg, want)
	}
}

func TestConfigureLoggingRefusesAnUnknownLevel(t *testing.T) {
	if err := yacypeerassembly.ConfigureLogging(envFrom(map[string]string{
		"LOG_LEVEL": "loud",
	})); err == nil {
		t.Error("logging configured with an unknown level, want an error")
	}
	if err := yacypeerassembly.ConfigureLogging(envFrom(nil)); err != nil {
		t.Errorf("ConfigureLogging: %v", err)
	}
}
