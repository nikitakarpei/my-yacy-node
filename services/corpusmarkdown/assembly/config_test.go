package corpusmarkdownassembly_test

import (
	"testing"

	corpusmarkdown "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/assembly"
	pagemarkdowncorporas3 "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pagemarkdowncorpora/s3"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func requiredEnv() map[string]string {
	return map[string]string{
		corpusmarkdown.EnvPageOfferNATSURL: "nats://crawl:4222",
	}
}

func objectStoreEnv() map[string]string {
	env := requiredEnv()
	env[corpusmarkdown.EnvPageMarkdownS3Endpoint] = "http://s3.internal:3900"
	env[corpusmarkdown.EnvPageMarkdownS3AccessKeyID] = "corpus"
	env[corpusmarkdown.EnvPageMarkdownS3SecretAccessKey] = "corpus-secret"
	return env
}

func TestLoadServiceConfigRequiresThePageOfferAddress(t *testing.T) {
	env := requiredEnv()
	delete(env, corpusmarkdown.EnvPageOfferNATSURL)
	if _, err := corpusmarkdown.LoadServiceConfig(envFrom(env)); err == nil {
		t.Errorf("expected error when %s is unset", corpusmarkdown.EnvPageOfferNATSURL)
	}
}

func TestLoadServiceConfigRequiresTheKeysOfANamedObjectStore(t *testing.T) {
	for _, missing := range []string{
		corpusmarkdown.EnvPageMarkdownS3AccessKeyID,
		corpusmarkdown.EnvPageMarkdownS3SecretAccessKey,
	} {
		env := objectStoreEnv()
		delete(env, missing)
		if _, err := corpusmarkdown.LoadServiceConfig(envFrom(env)); err == nil {
			t.Errorf("expected error when %s is unset", missing)
		}
	}
}

func TestLoadServiceConfigKeepsPageMarkdownInADirectoryWithoutAnObjectStore(t *testing.T) {
	cfg, err := corpusmarkdown.LoadServiceConfig(envFrom(requiredEnv()))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.PageMarkdownObjectStore != nil {
		t.Errorf("page markdown object store = %+v, want none", cfg.PageMarkdownObjectStore)
	}
	if cfg.PageMarkdownDirectory != corpusmarkdown.DefaultPageMarkdownDirectory {
		t.Errorf("page markdown directory = %q", cfg.PageMarkdownDirectory)
	}
}

func TestLoadServiceConfigRejectsADirectoryBesideAnObjectStore(t *testing.T) {
	env := objectStoreEnv()
	env[corpusmarkdown.EnvPageMarkdownDirectory] = "/var/lib/corpusmarkdown"
	if _, err := corpusmarkdown.LoadServiceConfig(envFrom(env)); err == nil {
		t.Error("a page markdown directory beside an object store should be rejected")
	}
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := corpusmarkdown.LoadServiceConfig(envFrom(objectStoreEnv()))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.PageOfferNATSURL != "nats://crawl:4222" {
		t.Errorf("page offer nats url = %q", cfg.PageOfferNATSURL)
	}
	want := pagemarkdowncorporas3.ObjectStore{
		Endpoint:        "http://s3.internal:3900",
		Region:          corpusmarkdown.DefaultPageMarkdownS3Region,
		AccessKeyID:     "corpus",
		SecretAccessKey: "corpus-secret",
	}
	if cfg.PageMarkdownObjectStore == nil || *cfg.PageMarkdownObjectStore != want {
		t.Errorf("page markdown object store = %+v, want %+v", cfg.PageMarkdownObjectStore, want)
	}
	if cfg.PageMarkdownBucket != corpusmarkdown.DefaultPageMarkdownBucket {
		t.Errorf("bucket = %q", cfg.PageMarkdownBucket)
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
	env := objectStoreEnv()
	env[corpusmarkdown.EnvPageOfferDurable] = "dur"
	env[corpusmarkdown.EnvPageOfferIntakeConcurrency] = "3"
	env[corpusmarkdown.EnvListenAddr] = "127.0.0.1:8099"
	env[corpusmarkdown.EnvOpsAddr] = "127.0.0.1:9099"
	env[corpusmarkdown.EnvPageMarkdownS3Region] = "garage"
	env[corpusmarkdown.EnvPageMarkdownBucket] = "markdown"

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
	if cfg.PageMarkdownObjectStore.Region != "garage" || cfg.PageMarkdownBucket != "markdown" {
		t.Errorf(
			"region/bucket = %q %q",
			cfg.PageMarkdownObjectStore.Region, cfg.PageMarkdownBucket,
		)
	}
}

func TestLoadServiceConfigTakesTheNamedPageMarkdownDirectory(t *testing.T) {
	env := requiredEnv()
	env[corpusmarkdown.EnvPageMarkdownDirectory] = "/var/lib/markdown"

	cfg, err := corpusmarkdown.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.PageMarkdownDirectory != "/var/lib/markdown" {
		t.Errorf("page markdown directory = %q", cfg.PageMarkdownDirectory)
	}
}

func TestLoadServiceConfigRejectsWhatItCannotRead(t *testing.T) {
	env := requiredEnv()
	env[corpusmarkdown.EnvPageOfferIntakeConcurrency] = "abc"
	if _, err := corpusmarkdown.LoadServiceConfig(envFrom(env)); err == nil {
		t.Error("a non-numeric intake concurrency should be rejected")
	}

	env = objectStoreEnv()
	env[corpusmarkdown.EnvPageMarkdownS3Endpoint] = "s3.internal:3900"
	if _, err := corpusmarkdown.LoadServiceConfig(envFrom(env)); err == nil {
		t.Error("an object store endpoint without a scheme should be rejected")
	}
}
