package rwi_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagerepresentations/rwi"
)

func TestRepresentationChunksBoundPostings(t *testing.T) {
	words := make([]string, 2001)
	for i := range words {
		words[i] = fmt.Sprintf("w%d", i)
	}
	representation := rwi.New()
	publication, err := representation.Frame(samplePage(), []byte(strings.Join(words, " ")))
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	messages := publication
	if len(messages) != 4 {
		t.Fatalf("messages = %d, want 4: metadata plus 1000, 1000 and 1 postings", len(messages))
	}
	for i, message := range messages {
		if _, err := yacycrawlcontract.UnmarshalPageRWIChunk(message); err != nil {
			t.Fatalf("unmarshal chunk %d: %v", i, err)
		}
	}
}

func TestRepresentationDeclaresKindAndContentFormat(t *testing.T) {
	representation := rwi.New()
	if representation.Kind() != yacycrawlcontract.PageRepresentationKindRWI {
		t.Fatalf("representation = %q", representation.Kind())
	}
	if representation.ContentFormat() != contentformatgraph.FormatFullText {
		t.Fatalf("content format = %q", representation.ContentFormat())
	}
}
