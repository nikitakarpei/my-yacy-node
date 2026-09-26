package htmlmeta_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta/htmlmeta"
)

const htmlContentType = "text/html; charset=utf-8"

func TestAPageWithoutRobotsMetaTagsRefusesNothing(t *testing.T) {
	refusals := htmlmeta.RefusalsOf(
		htmlContentType, []byte(`<html><body><a href="/next">next</a></body></html>`),
	)

	if refusals != (robotsmeta.Refusals{}) {
		t.Fatalf("refusals = %+v, want none", refusals)
	}
}

func TestEachRobotsMetaTagAddsItsRefusals(t *testing.T) {
	refusals := htmlmeta.RefusalsOf(htmlContentType, []byte(
		`<html><head><meta name="ROBOTS" content="noindex"/>`+
			`<meta name="robots" content="nofollow"></head></html>`,
	))

	if refusals != (robotsmeta.Refusals{RefusesIndexing: true, RefusesLinkDiscovery: true}) {
		t.Fatalf("refusals = %+v, want both", refusals)
	}
}

func TestAMetaTagForSomeoneElseStatesNoRefusal(t *testing.T) {
	refusals := htmlmeta.RefusalsOf(htmlContentType, []byte(
		`<html><head><meta name="googlebot" content="noindex">`+
			`<meta content="noindex"><meta name="robots"></head></html>`,
	))

	if refusals != (robotsmeta.Refusals{}) {
		t.Fatalf("refusals = %+v, want none", refusals)
	}
}

func TestABodyOfAnotherKindStatesNoRefusal(t *testing.T) {
	refusals := htmlmeta.RefusalsOf(
		"application/pdf", []byte(`<meta name="robots" content="noindex">`),
	)

	if refusals != (robotsmeta.Refusals{}) {
		t.Fatalf("refusals = %+v, want none", refusals)
	}
}

func TestAPageInAnotherCharsetStatesItsRefusals(t *testing.T) {
	refusals := htmlmeta.RefusalsOf("text/html; charset=utf-16le", []byte(
		"<\x00m\x00e\x00t\x00a\x00 \x00n\x00a\x00m\x00e\x00=\x00r\x00o\x00b\x00o\x00t\x00s\x00"+
			" \x00c\x00o\x00n\x00t\x00e\x00n\x00t\x00=\x00n\x00o\x00n\x00e\x00>\x00",
	))

	if refusals != (robotsmeta.Refusals{RefusesIndexing: true, RefusesLinkDiscovery: true}) {
		t.Fatalf("refusals = %+v, want both", refusals)
	}
}
