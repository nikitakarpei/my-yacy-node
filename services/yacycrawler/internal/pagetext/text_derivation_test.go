package pagetext_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagetext"
)

func TestDeriveTargetsTextFormat(t *testing.T) {
	if format := pagetext.New().Format(); format != crawlcapability.PageContentFormatText {
		t.Fatalf("format = %q, want text", format)
	}
}

func TestDeriveStripsMarkupFromHTML(t *testing.T) {
	body, err := pagetext.New().Derive(
		[]byte(`<article><h1>Title</h1><p>The quick <b>brown</b> fox.</p></article>`),
		crawlcapability.PageContentFormatHTML,
	)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	text := string(body)
	if strings.Contains(text, "<") {
		t.Fatalf("markup survived: %q", text)
	}
	if !strings.Contains(text, "The quick brown fox.") {
		t.Fatalf("inline markup should not split words: %q", text)
	}
}

func TestDeriveSeparatesBlockElements(t *testing.T) {
	body, err := pagetext.New().Derive(
		[]byte(`<p>first</p><p>second</p>`),
		crawlcapability.PageContentFormatHTML,
	)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if string(body) != "first\nsecond" {
		t.Fatalf("blocks not separated: %q", body)
	}
}

func TestDeriveDropsScriptAndStyle(t *testing.T) {
	body, err := pagetext.New().Derive(
		[]byte(`<p>keep</p><script>var drop = 1</script><style>.drop{}</style>`),
		crawlcapability.PageContentFormatHTML,
	)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if strings.Contains(string(body), "drop") {
		t.Fatalf("script or style content survived: %q", body)
	}
}

func TestDerivePassesTextThrough(t *testing.T) {
	body, err := pagetext.New().Derive(
		[]byte("already text"),
		crawlcapability.PageContentFormatText,
	)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if string(body) != "already text" {
		t.Fatalf("text body = %q", body)
	}
}

func TestDeriveRejectsUnsupportedSourceFormat(t *testing.T) {
	_, err := pagetext.New().Derive([]byte("x"), crawlcapability.PageContentFormat("audio"))
	if !errors.Is(err, crawlcapability.ErrUnsupportedSourceFormat) {
		t.Fatalf("err = %v, want ErrUnsupportedSourceFormat", err)
	}
}
