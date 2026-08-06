package main

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
)

func TestAdmittedMediaTypesDefaultsToTheWholeCatalog(t *testing.T) {
	admitted, err := admittedMediaTypesFor(ServiceConfig{MaxBodyBytes: 1 << 20})
	if err != nil {
		t.Fatalf("admit media types: %v", err)
	}
	if _, reads := admitted.extractors["text/html"]; !reads {
		t.Fatalf("html not admitted: %v", admitted.extractors)
	}
	if _, expands := admitted.containers["application/zip"]; !expands {
		t.Fatalf("zip not admitted: %v", admitted.containers)
	}
	if len(admitted.emittedFormats) != 1 ||
		admitted.emittedFormats[0] != contentformatgraph.FormatDocumentHTML {
		t.Fatalf("unexpected emitted formats: %v", admitted.emittedFormats)
	}
}

func TestAdmittedMediaTypesDropsTheExcludedExtractor(t *testing.T) {
	admitted, err := admittedMediaTypesFor(ServiceConfig{
		MaxBodyBytes: 1 << 20, ContentTypes: []string{"application/zip"},
	})
	if err != nil {
		t.Fatalf("admit media types: %v", err)
	}
	if len(admitted.extractors) != 0 {
		t.Fatalf("excluded extractor admitted: %v", admitted.extractors)
	}
	if len(admitted.emittedFormats) != 0 {
		t.Fatalf("excluded extractor still reports a format: %v", admitted.emittedFormats)
	}
}

func TestAdmittedMediaTypesRejectsUnregisteredContentType(t *testing.T) {
	if _, err := admittedMediaTypesFor(ServiceConfig{
		MaxBodyBytes: 1 << 20, ContentTypes: []string{"text/html", "application/unregistered"},
	}); err == nil {
		t.Fatal("a content type no extractor reads should fail startup")
	}
}
