package main

import (
	"net/http"
	"testing"
)

func TestLoadServiceConfigManticoreRequiresURL(t *testing.T) {
	if _, err := LoadServiceConfig(envFrom(map[string]string{
		EnvNATSURL:           "nats://localhost:4222",
		EnvSearchIndexEngine: SearchIndexEngineManticore,
	})); err == nil {
		t.Fatal("expected error when MANTICORE_URL is unset")
	}
}

func TestLoadServiceConfigManticoreDefaults(t *testing.T) {
	cfg, err := LoadServiceConfig(envFrom(map[string]string{
		EnvNATSURL:           "nats://localhost:4222",
		EnvSearchIndexEngine: SearchIndexEngineManticore,
		EnvManticoreURL:      "http://localhost:9308",
	}))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.ManticoreURL != "http://localhost:9308" {
		t.Errorf("manticore url = %q", cfg.ManticoreURL)
	}
	if cfg.ManticoreTable != DefaultIndexBaseName {
		t.Errorf("manticore table = %q", cfg.ManticoreTable)
	}
}

func TestSelectSearchIndexRejectsUnknownEngine(t *testing.T) {
	if _, err := selectSearchIndex(ServiceConfig{
		SearchIndexEngine: "sphinx",
	}, http.DefaultClient); err == nil {
		t.Fatal("expected error for unknown engine")
	}
}

func TestSelectSearchIndexRejectsAnUnsupportedLanguage(t *testing.T) {
	if _, err := selectSearchIndex(ServiceConfig{
		SearchIndexEngine:  SearchIndexEngineElasticsearch,
		ElasticsearchURL:   "http://localhost:9200",
		ElasticsearchIndex: DefaultIndexBaseName,
		Languages:          []string{"zz"},
	}, http.DefaultClient); err == nil {
		t.Fatal("expected error for an unsupported language")
	}
	if _, err := selectSearchIndex(ServiceConfig{
		SearchIndexEngine: SearchIndexEngineManticore,
		ManticoreURL:      "http://localhost:9308",
		ManticoreTable:    DefaultIndexBaseName,
		Languages:         []string{"zz"},
	}, http.DefaultClient); err == nil {
		t.Fatal("expected error for an unsupported language")
	}
}
