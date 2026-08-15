package mediatypeallowance_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/mediatypeallowance"
)

func TestAnEmptyContentTypeListAdmitsEveryMediaType(t *testing.T) {
	t.Parallel()

	allowance := mediatypeallowance.MediaTypeAllowanceFrom(nil)
	if !allowance.Admits("text/html") || !allowance.Admits("application/zip") {
		t.Fatal("an empty content type list should admit every registered media type")
	}
}

func TestAListedContentTypeIsTheOnlyOneAdmitted(t *testing.T) {
	t.Parallel()

	allowance := mediatypeallowance.MediaTypeAllowanceFrom([]string{"text/html"})
	if !allowance.Admits("text/html") {
		t.Fatal("a listed media type should be admitted")
	}
	if allowance.Admits("application/zip") {
		t.Fatal("a media type the operator did not list should be refused")
	}
}
