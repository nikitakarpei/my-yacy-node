package main

import (
	"testing"
	"time"
)

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

func TestLoadServiceConfigRequiresNATSURL(t *testing.T) {
	if _, err := LoadServiceConfig(envFrom(nil)); err == nil {
		t.Fatal("expected error when NATS_URL is unset")
	}
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	cfg, err := LoadServiceConfig(envFrom(map[string]string{
		EnvNATSURL: "nats://localhost:4222",
	}))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.OrdersSubject != DefaultOrdersSubject {
		t.Errorf("orders subject = %q", cfg.OrdersSubject)
	}
	if cfg.ListenAddr != DefaultListenAddr || cfg.OpsAddr != DefaultOpsAddr {
		t.Errorf("addrs = %q %q", cfg.ListenAddr, cfg.OpsAddr)
	}
	if cfg.Deadline != DefaultDeadline || cfg.PollInterval != DefaultPollInterval {
		t.Errorf("timings = %v %v", cfg.Deadline, cfg.PollInterval)
	}
	if cfg.MaxInFlight != DefaultMaxInFlight || cfg.MaxResponseBytes != DefaultMaxResponseBytes {
		t.Errorf("limits = %d %d", cfg.MaxInFlight, cfg.MaxResponseBytes)
	}
}

func TestLoadServiceConfigOverrides(t *testing.T) {
	cfg, err := LoadServiceConfig(envFrom(map[string]string{
		EnvNATSURL:          "nats://localhost:4222",
		EnvOrdersSubject:    "t.orders",
		EnvListenAddr:       "127.0.0.1:1000",
		EnvOpsAddr:          "127.0.0.1:1001",
		EnvDeadline:         "5s",
		EnvPollInterval:     "250ms",
		EnvMaxInFlight:      "8",
		EnvMaxResponseBytes: "1024",
	}))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.OrdersSubject != "t.orders" || cfg.ListenAddr != "127.0.0.1:1000" ||
		cfg.OpsAddr != "127.0.0.1:1001" {
		t.Errorf("strings = %+v", cfg)
	}
	if cfg.Deadline != 5*time.Second || cfg.PollInterval != 250*time.Millisecond {
		t.Errorf("timings = %v %v", cfg.Deadline, cfg.PollInterval)
	}
	if cfg.MaxInFlight != 8 || cfg.MaxResponseBytes != 1024 {
		t.Errorf("limits = %d %d", cfg.MaxInFlight, cfg.MaxResponseBytes)
	}
}

func TestLoadServiceConfigRejectsInvalidDeadline(t *testing.T) {
	if _, err := LoadServiceConfig(envFrom(map[string]string{
		EnvNATSURL:  "nats://localhost:4222",
		EnvDeadline: "soon",
	})); err == nil {
		t.Fatal("expected error for non-duration deadline")
	}
}

func TestLoadServiceConfigRejectsInvalidMaxInFlight(t *testing.T) {
	if _, err := LoadServiceConfig(envFrom(map[string]string{
		EnvNATSURL:     "nats://localhost:4222",
		EnvMaxInFlight: "-1",
	})); err == nil {
		t.Fatal("expected error for non-positive max in flight")
	}
}
