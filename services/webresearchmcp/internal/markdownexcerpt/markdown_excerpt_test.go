package markdownexcerpt_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/markdownexcerpt"
)

func TestMarkdownShorterThanTheLimitIsCarriedWhole(t *testing.T) {
	excerpt := markdownexcerpt.MarkdownExcerptOf("# Title", 100)

	if excerpt.Markdown != "# Title" {
		t.Errorf("markdown = %q, want the whole %q", excerpt.Markdown, "# Title")
	}
	if excerpt.MarkdownCharacterCount != 7 {
		t.Errorf("markdown character count = %d, want 7", excerpt.MarkdownCharacterCount)
	}
	if excerpt.Truncated {
		t.Error("excerpt says it is truncated, want the whole markdown")
	}
}

func TestMarkdownLongerThanTheLimitIsCutAndSaysHowLongTheWholeIs(t *testing.T) {
	excerpt := markdownexcerpt.MarkdownExcerptOf("abcdefghij", 4)

	if excerpt.Markdown != "abcd" {
		t.Errorf("markdown = %q, want the first four characters", excerpt.Markdown)
	}
	if excerpt.MarkdownCharacterCount != 10 {
		t.Errorf("markdown character count = %d, want the whole 10", excerpt.MarkdownCharacterCount)
	}
	if !excerpt.Truncated {
		t.Error("excerpt says it is whole, want it truncated")
	}
}

func TestTheLimitCountsCharactersAndNotBytes(t *testing.T) {
	excerpt := markdownexcerpt.MarkdownExcerptOf("äöüßé", 3)

	if excerpt.Markdown != "äöü" {
		t.Errorf("markdown = %q, want the first three characters", excerpt.Markdown)
	}
	if excerpt.MarkdownCharacterCount != 5 {
		t.Errorf("markdown character count = %d, want 5", excerpt.MarkdownCharacterCount)
	}
}
