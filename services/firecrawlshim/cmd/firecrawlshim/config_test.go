package main_test

import (
	"testing"

	firecrawlshim "github.com/nikitakarpei/yacy-rwi-node/firecrawlshim/cmd/firecrawlshim"
)

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadServiceConfigRequiresRecallTarget(t *testing.T) {
	if _, err := firecrawlshim.LoadServiceConfig(envFrom(nil)); err == nil {
		t.Fatal("expected error when recall target is unset")
	}
}

func TestLoadServiceConfigRejectsInvalidTimeout(t *testing.T) {
	_, err := firecrawlshim.LoadServiceConfig(envFrom(map[string]string{
		firecrawlshim.EnvRecallTarget:  "corpusrecall:8092",
		firecrawlshim.EnvRecallTimeout: "nope",
	}))
	if err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := firecrawlshim.LoadServiceConfig(envFrom(map[string]string{
		firecrawlshim.EnvRecallTarget: "corpusrecall:8092",
	}))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.ListenAddr != firecrawlshim.DefaultListenAddr {
		t.Errorf("listen addr = %q", cfg.ListenAddr)
	}
	if cfg.RecallTimeout != firecrawlshim.DefaultRecallTimeout {
		t.Errorf("recall timeout = %s", cfg.RecallTimeout)
	}
	if cfg.RecallTarget != "corpusrecall:8092" {
		t.Errorf("recall target = %q", cfg.RecallTarget)
	}
}
