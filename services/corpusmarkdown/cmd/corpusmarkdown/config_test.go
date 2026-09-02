package main_test

import (
	"testing"

	corpusmarkdown "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/cmd/corpusmarkdown"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func requiredEnv() map[string]string {
	return map[string]string{
		corpusmarkdown.EnvPageOfferNATSURL:    "nats://crawl:4222",
		corpusmarkdown.EnvPageMarkdownNATSURL: "nats://corpus:4222",
	}
}

func TestLoadServiceConfigRequiresEveryAddress(t *testing.T) {
	for _, missing := range []string{
		corpusmarkdown.EnvPageOfferNATSURL,
		corpusmarkdown.EnvPageMarkdownNATSURL,
	} {
		env := requiredEnv()
		delete(env, missing)
		if _, err := corpusmarkdown.LoadServiceConfig(envFrom(env)); err == nil {
			t.Errorf("expected error when %s is unset", missing)
		}
	}
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := corpusmarkdown.LoadServiceConfig(envFrom(requiredEnv()))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.PageOfferNATSURL != "nats://crawl:4222" {
		t.Errorf("page offer nats url = %q", cfg.PageOfferNATSURL)
	}
	if cfg.PageMarkdownNATSURL != "nats://corpus:4222" {
		t.Errorf("page markdown nats url = %q", cfg.PageMarkdownNATSURL)
	}
	if cfg.PageOfferDurable != corpusmarkdown.DefaultPageOfferDurable {
		t.Errorf("durable = %q", cfg.PageOfferDurable)
	}
	if cfg.PageOfferIntakeConcurrency != corpusmarkdown.DefaultPageOfferIntakeConcurrency {
		t.Errorf("pageOfferIntakeConcurrency = %d", cfg.PageOfferIntakeConcurrency)
	}
	if cfg.ListenAddr != corpusmarkdown.DefaultListenAddr {
		t.Errorf("listen addr = %q", cfg.ListenAddr)
	}
	if cfg.OpsAddr != corpusmarkdown.DefaultOpsAddr {
		t.Errorf("ops addr = %q", cfg.OpsAddr)
	}
}

func TestLoadServiceConfigOverrides(t *testing.T) {
	env := requiredEnv()
	env[corpusmarkdown.EnvPageOfferDurable] = "dur"
	env[corpusmarkdown.EnvPageOfferIntakeConcurrency] = "3"
	env[corpusmarkdown.EnvListenAddr] = "127.0.0.1:8099"
	env[corpusmarkdown.EnvOpsAddr] = "127.0.0.1:9099"

	cfg, err := corpusmarkdown.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.PageOfferDurable != "dur" {
		t.Errorf("durable = %q", cfg.PageOfferDurable)
	}
	if cfg.PageOfferIntakeConcurrency != 3 {
		t.Errorf("pageOfferIntakeConcurrency = %d", cfg.PageOfferIntakeConcurrency)
	}
	if cfg.ListenAddr != "127.0.0.1:8099" || cfg.OpsAddr != "127.0.0.1:9099" {
		t.Errorf("listen/ops addr = %q %q", cfg.ListenAddr, cfg.OpsAddr)
	}
}

func TestLoadServiceConfigRejectsWhatItCannotRead(t *testing.T) {
	env := requiredEnv()
	env[corpusmarkdown.EnvPageOfferIntakeConcurrency] = "abc"
	if _, err := corpusmarkdown.LoadServiceConfig(envFrom(env)); err == nil {
		t.Error("a non-numeric intake concurrency should be rejected")
	}
}
