package pagerwiassembly_test

import (
	"testing"

	pagerwi "github.com/nikitakarpei/yacy-rwi-node/pagerwi/assembly"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func requiredEnv() map[string]string {
	return map[string]string{
		pagerwi.EnvPageOfferNATSURL: "nats://scrape:4222",
		pagerwi.EnvPageRWINATSURL:   "nats://index:4222",
	}
}

func TestLoadServiceConfigRequiresEveryBroker(t *testing.T) {
	for _, missing := range []string{pagerwi.EnvPageOfferNATSURL, pagerwi.EnvPageRWINATSURL} {
		env := requiredEnv()
		delete(env, missing)
		if _, err := pagerwi.LoadServiceConfig(envFrom(env)); err == nil {
			t.Errorf("expected error when %s is unset", missing)
		}
	}
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := pagerwi.LoadServiceConfig(envFrom(requiredEnv()))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	want := pagerwi.ServiceConfig{
		PageOfferNATSURL:           "nats://scrape:4222",
		PageOfferDurable:           pagerwi.DefaultPageOfferDurable,
		PageOfferIntakeConcurrency: pagerwi.DefaultPageOfferIntakeConcurrency,
		PageRWINATSURL:             "nats://index:4222",
		PageRWIMaxBytes:            1 << 30,
		PageRWIPartitionExponent:   pagerwi.DefaultPageRWIPartitionExponent,
		ProvisionsStreams:          true,
		OpsAddr:                    pagerwi.DefaultOpsAddr,
	}
	if cfg != want {
		t.Errorf("config = %+v, want %+v", cfg, want)
	}
}

func TestLoadServiceConfigReadsEverySetting(t *testing.T) {
	env := requiredEnv()
	env[pagerwi.EnvPageOfferDurable] = "builders"
	env[pagerwi.EnvPageOfferIntakeConcurrency] = "8"
	env[pagerwi.EnvPageRWIMaxBytes] = "2MB"
	env[pagerwi.EnvPageRWIPartitionExponent] = "6"
	env[pagerwi.EnvProvisionStreams] = "false"
	env[pagerwi.EnvOpsAddr] = ":9191"

	cfg, err := pagerwi.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.PageOfferDurable != "builders" || cfg.PageOfferIntakeConcurrency != 8 ||
		cfg.PageRWIMaxBytes != 2<<20 || cfg.PageRWIPartitionExponent != 6 ||
		cfg.ProvisionsStreams || cfg.OpsAddr != ":9191" {
		t.Errorf("config = %+v, want every setting read", cfg)
	}
}

func TestLoadServiceConfigRejectsAnInvalidSetting(t *testing.T) {
	for key, value := range map[string]string{
		pagerwi.EnvPageOfferIntakeConcurrency: "0",
		pagerwi.EnvPageRWIMaxBytes:            "lots",
		pagerwi.EnvPageRWIPartitionExponent:   "-1",
		pagerwi.EnvProvisionStreams:           "sometimes",
	} {
		env := requiredEnv()
		env[key] = value
		if _, err := pagerwi.LoadServiceConfig(envFrom(env)); err == nil {
			t.Errorf("expected error for %s=%q", key, value)
		}
	}
}
