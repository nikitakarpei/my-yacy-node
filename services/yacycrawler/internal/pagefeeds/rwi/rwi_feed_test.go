package rwi

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
)

func TestRWIFeedChunksBoundPostings(t *testing.T) {
	words := make([]string, 2001)
	for i := range words {
		words[i] = fmt.Sprintf("w%d", i)
	}
	feed := New()
	publication, err := feed.Frame(samplePage(), []byte(strings.Join(words, " ")))
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	messages := publication.Messages()
	if len(messages) != 4 {
		t.Fatalf("messages = %d, want 4: metadata plus 1000, 1000 and 1 postings", len(messages))
	}
	for i, message := range messages {
		if _, err := yacycrawlcontract.UnmarshalPageRWIChunk(message); err != nil {
			t.Fatalf("unmarshal chunk %d: %v", i, err)
		}
	}
}

func TestRWIFeedDeclaresRepresentationAndContentFormat(t *testing.T) {
	feed := New()
	if feed.Representation() != yacycrawlcontract.PageRepresentationKindRWI {
		t.Fatalf("representation = %q", feed.Representation())
	}
	if feed.ContentFormat() != contentformatgraph.FormatFullText {
		t.Fatalf("content format = %q", feed.ContentFormat())
	}
}
