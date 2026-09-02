package main_test

import (
	"testing"

	corpustext "github.com/nikitakarpei/yacy-rwi-node/corpustext/cmd/corpustext"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func requiredEnv() map[string]string {
	return map[string]string{
		corpustext.EnvPageOfferNATSURL:  "nats://crawl:4222",
		corpustext.EnvSearchIndexEngine: corpustext.SearchIndexEngineElasticsearch,
		corpustext.EnvElasticsearchURL:  "http://localhost:9200",
	}
}

func TestLoadServiceConfigRequiresEveryAddress(t *testing.T) {
	for _, missing := range []string{
		corpustext.EnvPageOfferNATSURL,
		corpustext.EnvSearchIndexEngine,
		corpustext.EnvElasticsearchURL,
	} {
		env := requiredEnv()
		delete(env, missing)
		if _, err := corpustext.LoadServiceConfig(envFrom(env)); err == nil {
			t.Errorf("expected error when %s is unset", missing)
		}
	}
}

func TestLoadServiceConfigRejectsUnknownSearchIndexEngine(t *testing.T) {
	env := requiredEnv()
	env[corpustext.EnvSearchIndexEngine] = "sphinx"
	if _, err := corpustext.LoadServiceConfig(envFrom(env)); err == nil {
		t.Fatal("expected error for an unknown search index engine")
	}
}

func TestLoadServiceConfigManticoreRequiresURL(t *testing.T) {
	env := requiredEnv()
	env[corpustext.EnvSearchIndexEngine] = corpustext.SearchIndexEngineManticore
	if _, err := corpustext.LoadServiceConfig(envFrom(env)); err == nil {
		t.Fatal("expected error when MANTICORE_URL is unset")
	}
}

func TestLoadServiceConfigManticoreDefaults(t *testing.T) {
	env := requiredEnv()
	env[corpustext.EnvSearchIndexEngine] = corpustext.SearchIndexEngineManticore
	env[corpustext.EnvManticoreURL] = "http://localhost:9308"

	cfg, err := corpustext.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.ManticoreURL != "http://localhost:9308" {
		t.Errorf("manticore url = %q", cfg.ManticoreURL)
	}
	if cfg.ManticoreTable != corpustext.DefaultIndexBaseName {
		t.Errorf("manticore table = %q", cfg.ManticoreTable)
	}
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := corpustext.LoadServiceConfig(envFrom(requiredEnv()))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.PageOfferNATSURL != "nats://crawl:4222" {
		t.Errorf("page offer nats url = %q", cfg.PageOfferNATSURL)
	}
	if cfg.PageOfferDurable != corpustext.DefaultPageOfferDurable {
		t.Errorf("durable = %q", cfg.PageOfferDurable)
	}
	if cfg.PageOfferIntakeConcurrency != corpustext.DefaultPageOfferIntakeConcurrency {
		t.Errorf("pageOfferIntakeConcurrency = %d", cfg.PageOfferIntakeConcurrency)
	}
	if cfg.ElasticsearchIndex != corpustext.DefaultIndexBaseName {
		t.Errorf("index = %q", cfg.ElasticsearchIndex)
	}
	if cfg.OpsAddr != corpustext.DefaultOpsAddr {
		t.Errorf("ops addr = %q", cfg.OpsAddr)
	}
	if len(cfg.Languages) != 0 {
		t.Errorf("languages = %v", cfg.Languages)
	}
}

func TestLoadServiceConfigOverrides(t *testing.T) {
	env := requiredEnv()
	env[corpustext.EnvPageOfferDurable] = "dur"
	env[corpustext.EnvPageOfferIntakeConcurrency] = "3"
	env[corpustext.EnvElasticsearchIndex] = "my_index"
	env[corpustext.EnvLanguages] = "en, de"
	env[corpustext.EnvOpsAddr] = "127.0.0.1:9099"

	cfg, err := corpustext.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.PageOfferDurable != "dur" {
		t.Errorf("durable = %q", cfg.PageOfferDurable)
	}
	if cfg.PageOfferIntakeConcurrency != 3 || cfg.OpsAddr != "127.0.0.1:9099" {
		t.Errorf(
			"pageOfferIntakeConcurrency/ops addr = %d %q",
			cfg.PageOfferIntakeConcurrency,
			cfg.OpsAddr,
		)
	}
	if cfg.ElasticsearchIndex != "my_index" {
		t.Errorf("index = %q", cfg.ElasticsearchIndex)
	}
	if len(cfg.Languages) != 2 || cfg.Languages[0] != "en" || cfg.Languages[1] != "de" {
		t.Errorf("languages = %v", cfg.Languages)
	}
}

func TestLoadServiceConfigRejectsWhatItCannotRead(t *testing.T) {
	env := requiredEnv()
	env[corpustext.EnvPageOfferIntakeConcurrency] = "abc"
	if _, err := corpustext.LoadServiceConfig(envFrom(env)); err == nil {
		t.Error("a non-numeric intake concurrency should be rejected")
	}
}
