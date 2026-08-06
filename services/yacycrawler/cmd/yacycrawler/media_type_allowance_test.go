package main

import "testing"

func TestMediaTypeAllowanceAdmitsEverythingWhenUnset(t *testing.T) {
	if !mediaTypeAllowanceFrom(nil).admits("text/html") {
		t.Fatal("an empty content type list should admit every registered media type")
	}
}

func TestMediaTypeAllowanceAdmitsOnlyTheListedTypes(t *testing.T) {
	allowance := mediaTypeAllowanceFrom([]string{"text/html"})
	if !allowance.admits("text/html") || allowance.admits("application/zip") {
		t.Fatalf("unexpected allowance: %v", allowance)
	}
}

func TestEnsureRegisteredMediaTypesRejectsUnregisteredType(t *testing.T) {
	err := ensureRegisteredMediaTypes(
		[]string{"text/html", "application/unregistered"},
		map[string]bool{"text/html": true},
	)
	if err == nil {
		t.Fatal("a content type no extractor reads should fail startup")
	}
}
