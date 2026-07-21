package main

import "testing"

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestStartReturnsNonZeroOnInvalidConfig(t *testing.T) {
	origLookup := lookupEnv
	lookupEnv = envFrom(nil)
	defer func() { lookupEnv = origLookup }()

	if code := start(); code != 2 {
		t.Errorf("start() = %d, want 2", code)
	}
}

func TestLoadServiceConfigRequiresRecallTarget(t *testing.T) {
	if _, err := LoadServiceConfig(envFrom(nil)); err == nil {
		t.Fatal("expected error when recall target is unset")
	}
}

func TestLoadServiceConfigRejectsInvalidTimeout(t *testing.T) {
	_, err := LoadServiceConfig(envFrom(map[string]string{
		EnvRecallTarget:  "corpusrecall:8092",
		EnvRecallTimeout: "nope",
	}))
	if err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := LoadServiceConfig(envFrom(map[string]string{
		EnvRecallTarget: "corpusrecall:8092",
	}))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.ListenAddr != DefaultListenAddr {
		t.Errorf("listen addr = %q", cfg.ListenAddr)
	}
	if cfg.RecallTimeout != DefaultRecallTimeout {
		t.Errorf("recall timeout = %s", cfg.RecallTimeout)
	}
	if cfg.RecallTarget != "corpusrecall:8092" {
		t.Errorf("recall target = %q", cfg.RecallTarget)
	}
}
