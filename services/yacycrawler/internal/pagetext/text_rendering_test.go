package pagetext_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagetext"
)

func TestTextRenderingReadsTextAndTargetsText(t *testing.T) {
	rendering := pagetext.NewTextRendering()
	if source := rendering.SourceFormat(); source != crawlcapability.PageContentFormatText {
		t.Fatalf("source format = %q, want text", source)
	}
	if format := rendering.Format(); format != crawlcapability.PageContentFormatText {
		t.Fatalf("format = %q, want text", format)
	}
}

func TestTextRenderingPassesBodyThrough(t *testing.T) {
	body, err := pagetext.NewTextRendering().Render([]byte("already text"))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if string(body) != "already text" {
		t.Fatalf("text body = %q", body)
	}
}
